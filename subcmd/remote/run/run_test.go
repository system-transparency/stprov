package run

import (
	"net"
	"reflect"
	"testing"
)

func TestParseAllowedNets(t *testing.T) {
	for _, table := range []struct {
		desc  string
		addrs []string
		want  []net.IPNet
	}{
		{"invalid: ipv4: bad address", []string{"10.300.0.2"}, nil},
		{"invalid: ipv4: bad cidr", []string{"10.0.0.2/33"}, nil},
		{"invalid: ipv6: bad address", []string{"2001:db8::g1"}, nil},
		{"invalid: ipv6: bad cidr", []string{"2001:db8::1/129"}, nil},
		{"valid: ipv4", []string{"10.0.0.2/24", "10.0.1.3"}, ipNets(t, []string{"10.0.0.2/24", "10.0.1.3/32"})},
		{"valid: ipv6", []string{"2001:db8::1/64", "2001:db8:0:1::1"}, ipNets(t, []string{"2001:db8::1/64", "2001:db8:0:1::1/128"})},
	} {
		nets, err := parseAllowedNets(table.addrs)
		if got, want := err != nil, table.want == nil; got != want {
			t.Errorf("%s: got error %v but wanted %v: %v", table.desc, got, want, err)
			continue
		}
		if err != nil {
			continue
		}

		if got, want := nets, table.want; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got\n\t%v\nbut wanted\n\t%v", table.desc, got, want)
		}
	}
}

func ipNets(t *testing.T, addrs []string) (ret []net.IPNet) {
	for _, addr := range addrs {
		_, cidr, err := net.ParseCIDR(addr)
		if err != nil {
			t.Fatal(err)
		}
		ret = append(ret, *cidr)
	}
	return
}
