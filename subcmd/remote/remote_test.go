package remote

import (
	"time"

	"testing"
)

func TestCountTrue(t *testing.T) {
	for _, table := range []struct {
		in   []bool
		want int
	}{
		{[]bool{}, 0},
		{[]bool{false}, 0},
		{[]bool{true}, 1},
		{[]bool{false, false}, 0},
		{[]bool{false, true}, 1},
		{[]bool{true, false}, 1},
		{[]bool{true, true}, 2},
		{[]bool{false, true, false, false, true, false, false, false, true}, 3},
	} {
		if got, want := countTrue(table.in...), table.want; got != want {
			t.Errorf("countTrue failed for input %v: got %d, want %d",
				table.in, got, want)
		}
	}
}

func TestFormatDescription(t *testing.T) {
	timestamp, err := time.Parse(time.RFC3339, "2025-01-30T14:49:01+01:00")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := formatDescription("foo", timestamp),
		"stprov version foo; timestamp 2025-01-30T13:49:01Z"; got != want {
		t.Errorf("Unexpected description string, got %q, want %q", got, want)
	}
}

func TestDecodeSafeCIDR(t *testing.T) {
	for _, table := range []struct {
		in   string
		want string
	}{
		{"", ""},
		{"x", "x"},
		{"/", "/"},
		{"m", "/"},
		{"m/", "m/"},
		{"mm", "m/"},
		{"mm2", "m/2"},
		{"192.0.2.1/25", "192.0.2.1/25"},
		{"192.0.2.1m25", "192.0.2.1/25"},
		{"2001:db8::1/25", "2001:db8::1/25"},
		{"2001:db8::1m25", "2001:db8::1/25"},
	} {
		got := decodeSafeCIDR(table.in)

		if got != table.want {
			t.Errorf("Unexpected decoded string for input %s: got %s, want %s",
				table.in, got, table.want)
		}
	}
}
