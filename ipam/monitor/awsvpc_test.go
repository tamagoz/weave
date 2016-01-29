package monitor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/weaveworks/weave/net/address"
)

var (
	range1  = newRange("10.0.0.0", "10.0.0.255")
	range2  = newRange("10.0.0.0", "10.0.0.4")
	range3  = newRange("10.0.0.5", "10.0.0.255")
	range4  = newRange("10.0.0.2", "10.0.0.8")
	range5  = newRange("10.0.0.6", "10.0.0.10")
	range6  = newRange("10.0.0.12", "10.0.0.16")
	range7  = newRange("10.0.0.18", "10.0.0.20")
	range8  = newRange("10.0.0.22", "10.0.0.24")
	range9  = newRange("10.0.0.26", "10.0.0.30")
	range10 = newRange("10.0.0.28", "10.0.0.32")
	range11 = newRange("10.0.1.0", "10.0.1.255")
)

func newRange(start, end string) address.Range {
	ip := func(s string) address.Address {
		addr, _ := address.ParseIP(s)
		return addr
	}
	return address.NewRange(ip(start), address.Subtract(ip(end), ip(start)))
}

func TestGroupRangesNoChanges(t *testing.T) {
	oldRanges := []address.Range{range1}
	newRanges := []address.Range{range1}
	groups := groupRanges(oldRanges, newRanges)
	require.Equal(t, 0, len(groups), "")
}

func TestgroupsingleGap(t *testing.T) {
	oldRanges := []address.Range{range1}
	newRanges := []address.Range{range2, range3}
	groups := groupRanges(oldRanges, newRanges)
	require.Equal(t, 1, len(groups), "")
	require.Equal(t, 1, len(groups[0].old), "")
	require.Equal(t, range1, groups[0].old[0], "")
	require.Equal(t, 2, len(groups[0].new), "")
	require.Equal(t, range2, groups[0].new[0], "")
	require.Equal(t, range3, groups[0].new[1], "")
}

func TestGroupRangesGapAtBeginning(t *testing.T) {
	oldRanges := []address.Range{range1}
	newRanges := []address.Range{range3}
	groups := groupRanges(oldRanges, newRanges)
	require.Equal(t, 1, len(groups), "")
	require.Equal(t, 1, len(groups[0].old), "")
	require.Equal(t, range1, groups[0].old[0], "")
	require.Equal(t, 1, len(groups[0].new), "")
	require.Equal(t, range3, groups[0].new[0], "")
}

func TestGroupRangesGapAtEnd(t *testing.T) {
	oldRanges := []address.Range{range1}
	newRanges := []address.Range{range2}
	groups := groupRanges(oldRanges, newRanges)
	require.Equal(t, 1, len(groups), "")
	require.Equal(t, 1, len(groups[0].old), "")
	require.Equal(t, range1, groups[0].old[0], "")
	require.Equal(t, 1, len(groups[0].new), "")
	require.Equal(t, range2, groups[0].new[0], "")
}

func TestGroupRanges(t *testing.T) {
	oldRanges := []address.Range{range2, range5, range7, range8, range9}
	newRanges := []address.Range{range4, range6, range8, range10, range11}
	groups := groupRanges(oldRanges, newRanges)
	require.Equal(t, 5, len(groups), "")
	// 1st group
	require.Equal(t,
		rangeGroup{[]address.Range{range2, range5}, []address.Range{range4}},
		groups[0], "")
	// 2nd group
	require.Equal(t, rangeGroup{nil, []address.Range{range6}}, groups[1], "")
	// 3rd group
	require.Equal(t, rangeGroup{[]address.Range{range7}, nil}, groups[2], "")
	// 4th group
	require.Equal(t,
		rangeGroup{[]address.Range{range9}, []address.Range{range10}},
		groups[3], "")
	// 5th group
	require.Equal(t, rangeGroup{nil, []address.Range{range11}}, groups[4], "")
}
