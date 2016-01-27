package monitor

import (
	"github.com/weaveworks/weave/net/address"
)

type AwsVPCMonitor struct{}

func NewAwsVPCMonitor() *AwsVPCMonitor {
	return &AwsVPCMonitor{}
}

func (m *AwsVPCMonitor) HandleUpdate(oldRanges, newRanges []address.Range) {
	panic("NYI")
}
