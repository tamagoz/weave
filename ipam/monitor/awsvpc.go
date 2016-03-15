package monitor

// TODO docs

import (
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
func NewAwsVPCMonitor(routeTableID string) *AwsVPCMonitor {
	// TODO(mp) add detect mechanism for the routerTableId
	var err error
	session := session.New()
	mon := &AwsVPCMonitor{}

	if routeTableID == "" {
		common.Log.Fatalln("awsvpc: routeTableID cannot be empty")
	}
	mon.routeTableID = routeTableID

	// Detect host (peer) Instance ID and Region
	meta := ec2metadata.New(session)
	mon.instanceID, err = meta.GetMetadata("instance-id")
	if err != nil {
		common.Log.Fatalf("awsvpc: Cannot detect instance-id: %s\n", err)
	}
	region, err := meta.Region()
	if err != nil {
		common.Log.Fatalf("awsvpc: Cannot detect region: %s\n", err)
	}
	// Create EC2 session
	mon.ec2 = ec2.New(session, aws.NewConfig().WithRegion(region))

	common.Log.Infof("awsvpc: Successfully initialized. routeTableID: %s. instanceID: %s. region: %s\n",
		mon.routeTableID, mon.instanceID, region)

	// Detect Weave bridge link index
	// TODO(mp) pass as an argument bridge name
	link, err := netlink.LinkByName("weave")
	if err != nil {
		common.Log.Fatalln("awsvpc: Cannot find weave interface")
	}
	mon.linkIndex = link.Attrs().Index

	return mon
}

// HandleUpdate method updates routing table.
func (mon *AwsVPCMonitor) HandleUpdate(old, new []address.Range) {
	oldCIDRs, newCIDRs := filterSameCIDRs(address.NewCIDRs(old), address.NewCIDRs(new))
	for _, cidr := range newCIDRs {
		cidrStr := cidr.String()

		common.Log.Infof("awsvpc: Creating %s route to %s within %s route table.\n",
			cidr, mon.instanceID, mon.routeTableID)
		out, err := mon.createVPCRoute(cidrStr)
		if err != nil {
			common.Log.Fatalf("awsvpc: createVPCRoute: %s %s\n", err, out)
		}
		common.Log.Infof("awsvpc: Creating %s route on host.\n", cidr)
		err = mon.createHostRoute(cidrStr)
		if err != nil {
			common.Log.Fatalf("awsvpc: createHostRoute: %s\n", err)
		}
	}
	for _, cidr := range oldCIDRs {
		cidrStr := cidr.String()
		common.Log.Infof("awsvpc: Removing %s route from %s route table.\n",
			cidr, mon.routeTableID)
		out, err := mon.deleteVPCRoute(cidrStr)
		if err != nil {
			common.Log.Fatalf("awsvpc: deleteVPCRoute: %s %s\n", err, out)
		}
		common.Log.Infof("awsvpc: Removing %s route on host.\n", cidr)
		err = mon.deleteHostRoute(cidrStr)
		if err != nil {
			common.Log.Fatalf("awsvpc: deleteHostRoute: %s\n", err)
		}
	}
}

func (mon *AwsVPCMonitor) createVPCRoute(cidr string) (
	*ec2.CreateRouteOutput, error) {

	route := &ec2.CreateRouteInput{
		RouteTableId:         &mon.routeTableID,
		InstanceId:           &mon.instanceID,
		DestinationCidrBlock: &cidr,
	}

	return mon.ec2.CreateRoute(route)
}

func (mon *AwsVPCMonitor) createHostRoute(cidr string) error {
	dst, err := parseIP(cidr)
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
	dst, err := parseIP(cidr)
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

func filterSameCIDRs(old, new []address.CIDR) (filteredOld, filteredNew []address.CIDR) {
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

func parseIP(body string) (*net.IPNet, error) {
	ip, ipnet, err := net.ParseCIDR(string(body))
	if err != nil {
		return nil, err
	}
	ipnet.IP = ip

	return ipnet, nil
}
