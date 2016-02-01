package monitor

// TODO docs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/weaveworks/weave/common"
	"github.com/weaveworks/weave/net/address"
)

var (
	log = common.Log
)

type AwsVPCMonitor struct {
	ec2          *ec2.EC2
	instanceId   string
	routeTableId string
}

// NewAwsVPCMonitor creates and intializes AWS VPC based monitor.
//
// The monitor updates AWS VPC route table when any changes to allocated
// address ranges have been committed.
func NewAwsVPCMonitor(routeTableId string) *AwsVPCMonitor {
	// TODO(brb) add detect mechanism for the routerTableId
	var err error
	session := session.New()
	mon := &AwsVPCMonitor{}

	if routeTableId == "" {
		log.Fatalln("awsvpc: routeTableId cannot be empty")
	}
	mon.routeTableId = routeTableId

	// Detect host (peer) Instance ID and Region
	meta := ec2metadata.New(session)
	mon.instanceId, err = meta.GetMetadata("instance-id")
	if err != nil {
		log.Fatalf("awsvpc: Cannot detect instance-id: %s\n", err)
	}
	region, err := meta.Region()
	if err != nil {
		log.Fatalf("awsvpc: Cannot detect region: %s\n", err)
	}
	// Create EC2 session
	mon.ec2 = ec2.New(session, aws.NewConfig().WithRegion(region))

	log.Infof("awsvpc: Successfully initialized. routeTableId: %s. instanceId: %s. region: %s\n",
		mon.routeTableId, mon.instanceId, region)

	return mon
}

// HandleUpdate method updates routing table.
func (mon *AwsVPCMonitor) HandleUpdate(oldRanges, newRanges []address.Range) {
	for _, group := range groupRanges(oldRanges, newRanges) {
		// Create routes for new ranges
		for _, addr := range group.new {
			for _, cidr := range addr.CIDRs() {
				log.Infof("awsvpc: Creating %s route to %s within %s route table.\n",
					cidr, mon.instanceId, mon.routeTableId)
				mon.createRoute(cidr.String())
			}
		}
		// Delete old obsolete ranges
		for _, addr := range group.old {
			for _, cidr := range addr.CIDRs() {
				log.Infof("awsvpc: Removing %s route from %s route table.\n",
					cidr, mon.routeTableId)
				mon.deleteRoute(cidr.String())
			}
		}
	}
}

func (mon *AwsVPCMonitor) createRoute(cidr string) (
	*ec2.CreateRouteOutput, error) {

	route := &ec2.CreateRouteInput{
		RouteTableId:         &mon.routeTableId,
		InstanceId:           &mon.instanceId,
		DestinationCidrBlock: &cidr,
	}

	return mon.ec2.CreateRoute(route)
}

func (mon *AwsVPCMonitor) deleteRoute(cidr string) (*ec2.DeleteRouteOutput, error) {
	route := &ec2.DeleteRouteInput{
		RouteTableId:         &mon.routeTableId,
		DestinationCidrBlock: &cidr,
	}

	return mon.ec2.DeleteRoute(route)
}

// Helpers

// groupRanges function groups overlapping address ranges.
func groupRanges(oldRanges, newRanges []address.Range) []rangeGroup {
	groups := make([]rangeGroup, 0)
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
