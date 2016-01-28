package monitor

import (
	"os"

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
// The route table id should be accessible via the WEAVE_AWS_ROUTE_TABLE_ID
// environment variable.
func NewAwsVPCMonitor() *AwsVPCMonitor {
	var err error
	mon := &AwsVPCMonitor{}

	// Detect host (peer) Instance ID and Region
	meta := ec2metadata.New(nil)
	mon.instanceId, err = meta.GetMetadata("instance-id")
	if err != nil {
		log.Fatalf("Cannot detect instance-id: %s\n", err)
	}
	region, err := meta.Region()
	if err != nil {
		log.Fatalf("Cannot detect region: %s\n", err)
	}
	// Create EC2 session
	mon.ec2 = ec2.New(session.New(), aws.NewConfig().WithRegion(region))
	// Set Route Table ID
	// TODO(brb) add detect mechanism for the id
	mon.routeTableId = os.Getenv("WEAVE_AWS_ROUTE_TABLE_ID")
	if mon.routeTableId == "" {
		log.Fatalln("Please set WEAVE_AWS_ROUTE_TABLE_ID")
	}

	return mon
}

func (mon *AwsVPCMonitor) HandleUpdate(oldRanges, newRanges []address.Range) {
	panic("NYI")
}

func (mon *AwsVPCMonitor) createRoute(routeTableId, instanceId, cidr string) (*ec2.CreateRouteOutput, error) {
	route := &ec2.CreateRouteInput{
		RouteTableId:         &routeTableId,
		InstanceId:           &instanceId,
		DestinationCidrBlock: &cidr,
	}

	return mon.ec2.CreateRoute(route)
}

func (mon *AwsVPCMonitor) delRoute(routeTableId, cidr string) (*ec2.DeleteRouteOutput, error) {
	route := &ec2.DeleteRouteInput{
		RouteTableId:         &routeTableId,
		DestinationCidrBlock: &cidr,
	}

	return mon.ec2.DeleteRoute(route)
}
