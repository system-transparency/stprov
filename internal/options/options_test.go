package options

import (
	"flag"
	"fmt"
	"log"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
)

func Example() {
	macs, err := DefaultInterfaces(1 * time.Second)
	if err != nil {
		log.Printf("no interfaces with state UP")
	}
	log.Printf("mac addresses of interfaces with state UP: %v", macs)
	// Output:
}

func TestAddStringS(t *testing.T) {
	for _, table := range []struct {
		desc      string
		args      string
		defaultTo string
		want      []string
	}{
		{"default: empty string", "", "", nil},
		{"default: one value", "", "foo", []string{"foo"}},
		{"default: multiple values", "", "foo,bar", []string{"foo", "bar"}},
		{"set: one value", "-l foo", "foo,bar", []string{"foo"}},
		{"set: multiple values", "-l bar -l foo", "foo,bar", []string{"bar", "foo"}},
		{"set: comma-separated", "-l bar,foo", "foo,bar", []string{"bar", "foo"}},
	} {
		var got SliceFlag
		setOptions := func(fs *flag.FlagSet) {
			AddStringS(fs, &got, "l", "list", table.defaultTo)
		}
		usage := func() { fmt.Println("test-cmd is a unit test") }
		args := append([]string{"test-cmd"}, strings.Split(table.args, " ")...)

		New(args, usage, setOptions)
		if got, want := got.Values, table.want; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v but wanted %v", table.desc, got, want)
		}
	}
}

func TestParseProvisioningURL(t *testing.T) {
	for _, table := range []struct {
		desc    string
		absURL  string
		tmplURL string
		user    string
		pass    string
		want    string
	}{
		{"invalid: prefix", "example.org", "", "", "", ""},
		{"invalid: require user", "", "", "", "", ""},
		{"invalid: require password", "", "", "1234", "", ""},
		{"invalid: require no user", "http://example.org", "", "alice", "", ""},
		{"invalid: require no password", "http://example.org", "", "", "1234", ""},
		{"valid", "http://example.org", "", "", "", "http://example.org"},
		{"valid", "https://example.org", "", "", "", "https://example.org"},
		{"valid", "", "https://user:password@example.org", "alice", "1234", "https://alice:1234@example.org"},
	} {
		url, err := ParseProvisioningURL(table.absURL, table.tmplURL, table.user, table.pass)
		if got, want := err != nil, table.desc != "valid"; got != want {
			t.Errorf("%q: got error %v but wanted %v: %v", table.desc, got, want, err)
		}
		if err != nil {
			continue
		}
		if got, want := url, table.want; got != want {
			t.Errorf("%q: got url %s but wanted %s", table.desc, got, want)
		}
	}
}

var cases = []struct {
	test     string
	expected string
}{
	{
		test:     "10.0.2.15/32",
		expected: "10.0.2.15",
	},
	{
		test:     "10.0.2.15/31",
		expected: "10.0.2.15",
	},
	{
		test:     "10.0.2.15/27",
		expected: "10.0.2.30",
	},
	{
		test:     "2001:db8::/34",
		expected: "2001:db8:3fff:ffff:ffff:ffff:ffff:fffe",
	},
	{
		test:     "2001:db8::/128",
		expected: "2001:db8::",
	},
	{
		test:     "2001:db8::/122",
		expected: "2001:db8::3e",
	},
}

func stoip(s string) *net.IPNet {
	_, network, _ := net.ParseCIDR(s)
	return network
}

func TestMaxHost(t *testing.T) {
	for n, test := range cases {
		r := MaxHost(stoip(test.test))
		if r != test.expected {
			t.Errorf("failed case %d, expected %s got %s", n, test.expected, r)
		}
	}
}
