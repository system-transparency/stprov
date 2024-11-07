package remote

import (
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
