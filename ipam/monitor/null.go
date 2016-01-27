package monitor

import (
	"github.com/weaveworks/weave/net/address"
)

type NullMonitor struct{}

func NewNullMonitor() *NullMonitor {
	return &NullMonitor{}
}

func (m *NullMonitor) HandleUpdate(oldRanges, newRanges []address.Range) {
	// do nothing
}
