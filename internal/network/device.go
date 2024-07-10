package network

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/vishvananda/netlink"
)

func WaitForDeviceEvent(ctx context.Context, iface string, state netlink.LinkOperState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	links := make(chan netlink.LinkUpdate)
	if err := netlink.LinkSubscribe(links, done); err != nil {
		return fmt.Errorf("linksubscribe failed: %w", err)
	}
	defer close(done)
	select {
	case event := <-links:
		log.Printf("got link update on %s, in operstate %s", event.Attrs().Name, event.Attrs().OperState.String())
		if event.Attrs().Name == iface && event.Attrs().OperState == state {
			return nil
		}
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")

	}
	return nil
}

// Taken from the linux kernel
// https://github.com/torvalds/linux/blob/v6.0/drivers/net/phy/phy-core.c#L14
func speedToStr(speed string) string {
	switch speed {
	case "10":
		return "10Mbps"
	case "100":
		return "100Mbps"
	case "1000":
		return "1Gbps"
	case "2500":
		return "2.5Gbps"
	case "5000":
		return "5Gbps"
	case "10000":
		return "10Gbps"
	case "14000":
		return "14Gbps"
	case "20000":
		return "20Gbps"
	case "25000":
		return "25Gbps"
	case "40000":
		return "40Gbps"
	case "50000":
		return "50Gbps"
	case "56000":
		return "56Gbps"
	case "100000":
		return "100Gbps"
	case "unknown":
		return "Unknown"
	case "-1":
		return "Unknown"
	default:
		return "Unsupported (update stprov)"
	}
}

func GetDeviceSpeed(device string) string {
	base := path.Join("/sys/class/net", device)
	f := path.Join(base, "speed")
	b, err := os.ReadFile(f)
	if err != nil {
		return "Unknown"
	}
	return speedToStr(strings.TrimSpace(string(b)))
}

func GetDeviceDuplex(device string) string {
	base := path.Join("/sys/class/net", device)
	f := path.Join(base, "duplex")
	b, err := os.ReadFile(f)
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(b))
}
