package network

import (
	"slices"
	"testing"

	"github.com/vishvananda/netlink"
)

type testLink struct {
	attrs netlink.LinkAttrs
}

func (l *testLink) Attrs() *netlink.LinkAttrs {
	return &l.attrs
}
func (l *testLink) Type() string {
	return "test"
}

func TestLinksByDescendingSpeed(t *testing.T) {
	for _, table := range []struct {
		names  []string
		speeds []int64
		want   []string
	}{
		{
			names:  nil,
			speeds: nil,
			want:   nil,
		},
		{
			names:  []string{"a"},
			speeds: []int64{10},
			want:   []string{"a"},
		},
		{
			names:  []string{"a", "b", "c", "d"},
			speeds: []int64{10, 10, -1, 100},
			want:   []string{"d", "a", "b", "c"},
		},
	} {
		devices := make([]linkWithSpeed, len(table.names))
		for i, name := range table.names {
			devices[i] = linkWithSpeed{
				link:          &testLink{attrs: netlink.LinkAttrs{Name: name}},
				bitsPerSecond: table.speeds[i],
			}
		}
		links := linksByDescendingSpeed(devices)
		got := make([]string, len(links))
		for i, link := range links {
			got[i] = link.Attrs().Name
		}
		if !slices.Equal(got, table.want) {
			t.Errorf("failed for input devices %v, speeds %v: got %v, want %v",
				table.names, table.speeds, got, table.want)
		}
	}
}
