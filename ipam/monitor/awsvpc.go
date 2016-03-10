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

var (
	log = common.Log
)

type AwsVPCMonitor struct {
	ec2          *ec2.EC2
	instanceID   string
	routeTableID string
	linkIndex    int
}

// NewAwsVPCMonitor creates and intializes AWS VPC based monitor.
//
// The monitor updates AWS VPC and host route tables when any changes to allocated
// address ranges have been committed.
func NewAwsVPCMonitor(routeTableID string) *AwsVPCMonitor {
	// TODO(brb) add detect mechanism for the routerTableId
	var err error
	session := session.New()
	mon := &AwsVPCMonitor{}

	if routeTableID == "" {
		log.Fatalln("awsvpc: routeTableID cannot be empty")
	}
	mon.routeTableID = routeTableID

	// Detect host (peer) Instance ID and Region
	meta := ec2metadata.New(session)
	mon.instanceID, err = meta.GetMetadata("instance-id")
	if err != nil {
		log.Fatalf("awsvpc: Cannot detect instance-id: %s\n", err)
	}
	region, err := meta.Region()
	if err != nil {
		log.Fatalf("awsvpc: Cannot detect region: %s\n", err)
	}
	// Create EC2 session
	mon.ec2 = ec2.New(session, aws.NewConfig().WithRegion(region))

	log.Infof("awsvpc: Successfully initialized. routeTableID: %s. instanceID: %s. region: %s\n",
		mon.routeTableID, mon.instanceID, region)

	// Detect Weave bridge link index
	// TODO(brb) pass as an argument bridge name
	link, err := netlink.LinkByName("weave")
	if err != nil {
		log.Fatalln("awsvpc: Cannot find weave interface")
	}
	mon.linkIndex = link.Attrs().Index

	return mon
}

// HandleUpdate method updates routing table.
func (mon *AwsVPCMonitor) HandleUpdate(oldRanges, newRanges []address.Range) {
	for _, group := range groupRanges(oldRanges, newRanges) {
		// Create routes for new ranges
		for _, addr := range group.new {
			for _, cidr := range addr.CIDRs() {
				log.Infof("awsvpc: Creating %s route to %s within %s route table.\n",
					cidr, mon.instanceID, mon.routeTableID)
				out, err := mon.createVPCRoute(cidr.String())
				if err != nil {
					log.Fatalf("awsvpc: createVPCRoute: %s %s\n", err, out)
				}
				log.Infof("awsvpc: Creating %s route on host.\n", cidr)
				err = mon.createHostRoute(cidr.String())
				if err != nil {
					log.Fatalf("awsvpc: createHostRoute: %s\n", err)
				}
			}
		}
		// Delete old obsolete ranges
		for _, addr := range group.old {
			for _, cidr := range addr.CIDRs() {
				log.Infof("awsvpc: Removing %s route from %s route table.\n",
					cidr, mon.routeTableID)
				out, err := mon.deleteRoute(cidr.String())
				if err != nil {
					log.Fatalf("awsvpc: deleteRoute: %s %s\n", err, out)
				}
				log.Infof("awsvpc: Removing %s route on host.\n", cidr)
				err = mon.deleteHostRoute(cidr.String())
				if err != nil {
					log.Fatalf("awsvpc: deleteHostRoute: %s\n", err)
				}
			}
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

func (mon *AwsVPCMonitor) deleteRoute(cidr string) (*ec2.DeleteRouteOutput, error) {
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

// groupRanges function groups overlapping address ranges.
// TODO(mp) just find diff instead of performing the grouping.
// TODO(mp) move to tmp
func groupRanges(oldRanges, newRanges []address.Range) []rangeGroup {
	var groups []rangeGroup
	// i for oldRanges, j for newRanges
	i, j := 0, 0
	fromi, fromj := i, j
	overlapped := false

	for i < len(oldRanges) && j < len(newRanges) {
		if oldRanges[i].Start == newRanges[j].Start &&
			oldRanges[i].End == newRanges[j].End {
			// Ranges match, continue
			i++
			j++
		} else if oldRanges[i].Overlaps(newRanges[j]) {
			// Ranges overlap, so keep iterating by incrementing index counter
			// of a range which ends before.
			if !overlapped {
				// Set from index counters
				fromi, fromj = i, j
			}
			overlapped = true
			if oldRanges[i].End >= newRanges[j].End {
				j++
			} else {
				i++
			}
		} else {
			// Ranges do not overlap. Check whether they have overlapped
			// in the previous iteration.
			if overlapped {
				// Restore index counter.
				if oldRanges[i].End < newRanges[j].End {
					j--
				} else {
					i--
				}
				// Append.
				appendToGroups(fromi, fromj, i, j, &oldRanges, &newRanges, &groups)
				i++
				j++
			} else {
				// They did not overlap before, so create an entry.
				if oldRanges[i].End < newRanges[j].End {
					groups = append(groups, rangeGroup{[]address.Range{oldRanges[i]}, nil})
					i++
				} else {
					groups = append(groups, rangeGroup{nil, []address.Range{newRanges[j]}})
					j++
				}
			}
			overlapped = false
		}
	}
	// Finished all iteration. Check whether previous elements overlapped. If
	// yes, then create a new group.
	if overlapped {
		if i == len(oldRanges) {
			i--
		} else {
			j--
		}
		appendToGroups(fromi, fromj, i, j, &oldRanges, &newRanges, &groups)
		i++
		j++
	}
	// Add the rest to separate groups, because the rest does not overlap.
	if size := len(oldRanges); i != size {
		rest := group(i, size-1, &oldRanges)
		groups = append(groups, rangeGroup{rest, nil})
	}
	if size := len(newRanges); j != size {
		rest := group(j, size-1, &newRanges)
		groups = append(groups, rangeGroup{nil, rest})
	}

	return groups
}

func appendToGroups(fromOld, fromNew, toOld, toNew int,
	oldRanges, newRanges *[]address.Range, groups *[]rangeGroup) {

	oldGroup := group(fromOld, toOld, oldRanges)
	newGroup := group(fromNew, toNew, newRanges)
	*groups = append(*groups, rangeGroup{oldGroup, newGroup})
}

func group(from, to int, r *[]address.Range) []address.Range {
	tmp := make([]address.Range, to-from+1)
	copy(tmp, (*r)[from:to+1])
	return tmp
}

type rangeGroup struct {
	old, new []address.Range
}

func parseIP(body string) (*net.IPNet, error) {
	ip, ipnet, err := net.ParseCIDR(string(body))
	if err != nil {
		return nil, err
	}
	ipnet.IP = ip

	return ipnet, nil
}
