package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/go-ping/ping"
	"github.com/vishvananda/netlink"
)

// ForEachInterface runs a callback on all non-loopback interfaces
func ForEachInterface(f func(link netlink.Link) error) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed accessing interfaces: %v", err)
	}
	for _, i := range interfaces {
		if i.Flags&net.FlagLoopback != 0 {
			continue
		}
		link, err := netlink.LinkByName(i.Name)
		if err != nil {
			return err
		}
		if err := f(link); err != nil {
			return err
		}
	}
	return nil
}

type Pinger struct {
	*ping.Pinger

	ctx       context.Context
	ctxCancel context.CancelFunc
}

func NewPinger(addr string) (*Pinger, error) {
	pinger, err := ping.NewPinger(addr)
	if err != nil {
		return nil, err
	}
	pinger.SetPrivileged(true)
	return &Pinger{Pinger: pinger}, nil
}

func (p *Pinger) Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	p.ctx, p.ctxCancel = context.WithCancel(ctx)
	defer p.ctxCancel()

	go func() {
		<-ctx.Done()
		p.Pinger.Stop()
	}()

	return p.Pinger.Run()
}

func (p *Pinger) Stop() {
	p.ctxCancel()
}

// try send 3 icmp packets to some ip over 3 seconds
func testGateway(gw *net.IP) error {
	pinger, err := NewPinger(gw.String())
	if err != nil {
		return err
	}
	pinger.Pinger.Count = 3
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	err = pinger.Run(ctx)
	if err != nil {
		return err
	}
	if pinger.Pinger.Statistics().PacketsRecv == 0 {
		return fmt.Errorf("couldn't ping gateway")
	}
	return nil
}

// Configurer a link with the gateway
func confLink(link netlink.Link, gw *net.IP, addr *netlink.Addr) error {
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("failed addradd: %w", err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed linksetup: %w", err)
	}
	r := &netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: link.Attrs().Index,
		Gw:        *gw,
		Priority:  100,
	}
	if err := netlink.RouteAdd(r); err != nil {
		return fmt.Errorf("failed routeadd: %w", err)
	}
	return nil
}

func hasAddrs(link netlink.Link) bool {
	addrs, _ := netlink.AddrList(link, netlink.FAMILY_ALL)
	return len(addrs) != 0
}

func ResetInterfaces() error {
	return ForEachInterface(func(link netlink.Link) error {
		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			return fmt.Errorf("%s: failed accessing address: %v", link.Attrs().Name, err)
		}
		for _, addr := range addrs {
			if err := netlink.AddrDel(link, &addr); err != nil {
				return fmt.Errorf("%s: failed resetting address: %v", link.Attrs().Name, err)
			}
		}
		if err := netlink.LinkSetDown(link); err != nil {
			return fmt.Errorf("%s: set link down: %v", link.Attrs().Name, err)
		}
		return nil
	})
}

func TestInterfaces(gw, addr string, interfaceWait time.Duration) ([]netlink.Link, error) {
	var testedDevices []netlink.Link
	gwIP := net.ParseIP(gw)
	addrIP, err := netlink.ParseAddr(addr)
	if err != nil {
		return nil, err
	}
	err = ForEachInterface(func(link netlink.Link) error {
		// Skip bonding interfaces
		if strings.HasPrefix(link.Attrs().Name, "bond") {
			return nil
		}
		log.Printf("testing link %s with MAC %s...", link.Attrs().Name, link.Attrs().HardwareAddr)
		if hasAddrs(link) {
			log.Println("has addrs. Skipping.")
			return nil
		}
		if err := confLink(link, &gwIP, addrIP); err != nil {
			return fmt.Errorf("failed to configure link %s: %w", link.Attrs().Name, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), interfaceWait)
		defer cancel()
		WaitForDeviceEvent(ctx, link.Attrs().Name, netlink.OperUp)
		if err := testGateway(&gwIP); err != nil {
			log.Println(err)
		} else {
			duplex := GetDeviceDuplex(link.Attrs().Name)
			speed := GetDeviceSpeed(link.Attrs().Name)
			log.Printf("link is available! speed: %s duplex: %s\n", speed, duplex)
			testedDevices = append(testedDevices, link)
		}
		ResetInterfaces()
		return nil
	})
	return testedDevices, err
}
