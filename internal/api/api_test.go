package api

import (
	"bytes"
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := testServer(t)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Run(ctx); err != nil {
			t.Errorf("server failed: %v", err)
		}
	}()

	cli := testClient(t)

	// We race with the server startup, and may get here before
	// the server goroutine calls listen(2). There's seems to be
	// no way to get a signal from http.Server when it's ready to
	// serve requests, so instead, just retry a few times.
	var data *AddDataRequest
	try := 0
	for {
		var err error
		data, err = cli.AddData()
		if err == nil {
			break
		}
		try++
		if try >= 3 {
			t.Fatalf("client add-data failed: %v", err)
		}
		t.Logf("client add-data failed: %v, retrying in 1s", err)
		time.Sleep(time.Second)
	}
	cr, err := cli.Commit()
	if err != nil {
		t.Errorf("client commit failed: %v", err)
	}

	wg.Wait()
	if got, want := srv.Timestamp, data.Timestamp; got != want {
		t.Errorf("got timestamp %d but wanted %d", got, want)
	}
	if got, want := srv.Entropy[:], data.Entropy; !bytes.Equal(got, want) {
		t.Errorf("got entropy\n%v\nbut wanted\n%v", got, want)
	}
	if got, want := cr.HostName, srv.HostName; got != want {
		t.Errorf("got host name %q but wanted %q", got, want)
	}
}

func testServer(t *testing.T) *Server {
	t.Helper()
	secret, allowedNets, ip, port := testParams(t)
	srv, err := NewServer(&ServerConfig{
		Secret:     secret,
		LocalCIDR:  allowedNets,
		RemoteIP:   ip,
		RemotePort: port,
		HostName:   "mullis",
		Deadline:   5 * time.Second,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func testClient(t *testing.T) *Client {
	t.Helper()
	secret, _, ip, port := testParams(t)
	cli, err := NewClient(&ClientConfig{
		Secret:     secret,
		RemoteIP:   ip,
		RemotePort: port,
	})
	if err != nil {
		t.Fatal(err)
	}
	return cli
}

func testParams(t *testing.T) (secret string, allowedNets []net.IPNet, ip net.IP, port int) {
	t.Helper()
	var err error
	secret = "red"
	_, cidr1, err := net.ParseCIDR("127.0.0.1/25")
	if err != nil {
		t.Fatal(err)
	}
	_, cidr2, err := net.ParseCIDR("10.0.0.1/25")
	if err != nil {
		t.Fatal(err)
	}
	allowedNets = []net.IPNet{*cidr1, *cidr2}
	ip = net.IPv4(127, 0, 0, 1)
	port = 2009
	return
}
