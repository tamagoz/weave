package monitor

// TODO(mp) docs

import (
	"fmt"
	"net"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/vishvananda/netlink"

	"github.com/weaveworks/weave/common"
	"github.com/weaveworks/weave/net/address"
)

type AwsVPCMonitor struct {
	ec2          *ec2.EC2
	instanceID   string
	routeTableID string
	linkIndex    int
}

// NewAwsVPCMonitor creates and initialises AWS VPC based monitor.
//
// The monitor updates AWS VPC and host route tables when any changes to allocated
// address ranges owner by a peer have been committed.
func NewAwsVPCMonitor(routeTableID string) (*AwsVPCMonitor, error) {
	// TODO(mp) add detect mechanism for the routerTableId
	var err error
	session := session.New()
	mon := &AwsVPCMonitor{}

	if routeTableID == "" {
		return nil, fmt.Errorf("routeTableID cannot be empty")
	}
	mon.routeTableID = routeTableID

	// Detect host (peer) Instance ID and Region
	meta := ec2metadata.New(session)
	mon.instanceID, err = meta.GetMetadata("instance-id")
	if err != nil {
		return nil, fmt.Errorf("cannot detect instance-id due to %s", err)
	}
	region, err := meta.Region()
	if err != nil {
		return nil, fmt.Errorf("cannot detect region due to %s", err)
	}
	// Create EC2 session
	mon.ec2 = ec2.New(session, aws.NewConfig().WithRegion(region))

	// Detect Weave bridge link index
	// TODO(mp) pass as an argument bridge name
	bridgeIfName := "weave"
	link, err := netlink.LinkByName(bridgeIfName)
	if err != nil {
		return nil, fmt.Errorf("cannot find \"%s\" interface", bridgeIfName)
	}
	mon.linkIndex = link.Attrs().Index

	common.Log.Debugf(
		"AWSVPC monitor has been initialized on %s instance for %s route table at %s region",
		mon.instanceID, mon.routeTableID, region)

	return mon, nil
}

// HandleUpdate method updates the AWS VPC and the host route tables.
func (mon *AwsVPCMonitor) HandleUpdate(old, new []address.Range) error {
	oldCIDRs, newCIDRs := filterOutSameCIDRs(address.NewCIDRs(old), address.NewCIDRs(new))

	// It might make sense to do removal first and then add entries
	// because of the 50 routes limit. However, in such case a container might
	// not be reachable for short period of time which we we would like to
	// avoid.

	// Add new entries
	for _, cidr := range newCIDRs {
		cidrStr := cidr.String()
		common.Log.Debugf("Creating %s route to %s.", cidrStr, mon.instanceID)
		out, err := mon.createVPCRoute(cidrStr)
		// TODO(mp) check for 50 routes limit
		// TODO(mp) maybe check for auth related errors
		if err != nil {
			return fmt.Errorf("createVPCRoutes failed due to %s; details: %s",
				err, out)
		}
		err = mon.createHostRoute(cidrStr)
		if err != nil {
			return fmt.Errorf("createHostRoute failed due to %s", err)
		}
	}

	// Remove obsolete entries
	for _, cidr := range oldCIDRs {
		cidrStr := cidr.String()
		common.Log.Debugf("Removing %s route", cidrStr)
		out, err := mon.deleteVPCRoute(cidrStr)
		if err != nil {
			return fmt.Errorf("deleteVPCRoute failed due to %s; details: %s",
				err, out)
		}
		err = mon.deleteHostRoute(cidrStr)
		if err != nil {
			return fmt.Errorf("deleteHostRoute failed due to %s", err)
		}
	}

	return nil
}

func (mon *AwsVPCMonitor) createVPCRoute(cidr string) (*ec2.CreateRouteOutput, error) {
	route := &ec2.CreateRouteInput{
		RouteTableId:         &mon.routeTableID,
		InstanceId:           &mon.instanceID,
		DestinationCidrBlock: &cidr,
	}
	return mon.ec2.CreateRoute(route)
}

func (mon *AwsVPCMonitor) createHostRoute(cidr string) error {
	dst, err := parseCIDR(cidr)
	if err != nil {
		return err
	}
	route := &netlink.Route{
		LinkIndex: mon.linkIndex,
		Dst:       dst,
	}
	return netlink.RouteAdd(route)
}

func (mon *AwsVPCMonitor) deleteVPCRoute(cidr string) (*ec2.DeleteRouteOutput, error) {
	route := &ec2.DeleteRouteInput{
		RouteTableId:         &mon.routeTableID,
		DestinationCidrBlock: &cidr,
	}
	return mon.ec2.DeleteRoute(route)
}

func (mon *AwsVPCMonitor) deleteHostRoute(cidr string) error {
	dst, err := parseCIDR(cidr)
	if err != nil {
		return err
	}
	route := &netlink.Route{
		LinkIndex: mon.linkIndex,
		Dst:       dst,
	}
	return netlink.RouteDel(route)
}

// Helpers

// filterOutSameCIDRs filters out CIDR ranges which are contained in both new
// and old slices.
func filterOutSameCIDRs(old, new []address.CIDR) (filteredOld, filteredNew []address.CIDR) {
	i, j := 0, 0
	for i < len(old) && j < len(new) {
		switch {
		case old[i].Start() == new[j].Start() && old[i].End() == new[j].End():
			i++
			j++
			continue
		case old[i].End() < new[j].End():
			filteredOld = append(filteredOld, old[i])
			i++
		default:
			filteredNew = append(filteredNew, new[j])
			j++
		}
	}
	filteredOld = append(filteredOld, old[i:]...)
	filteredNew = append(filteredNew, new[j:]...)

	return filteredOld, filteredNew
}

func parseCIDR(cidr string) (*net.IPNet, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	ipnet.IP = ip

	return ipnet, nil
}
