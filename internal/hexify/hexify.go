package hexify

import (
	"bytes"
	"fmt"
)

// Format formats a buffer as a pretty pretty hex string, source:
// https://gist.github.com/chmike/05da938833328a9a94e02506922f2e7b
func Format(in []byte) string {
	out := bytes.NewBuffer(nil)
	buf := [16]byte{}
	n := (len(in) + 15) &^ 15
	for i := 0; i < n; i++ {
		if i%16 == 0 {
			out.WriteString(fmt.Sprintf("%4d", i))
		}
		if i%8 == 0 {
			out.WriteString(fmt.Sprint(" "))
		}
		if i < len(in) {
			out.WriteString(fmt.Sprintf(" %02X", in[i]))
		} else {
			out.WriteString(fmt.Sprint("   "))
		}
		if i >= len(in) {
			buf[i%16] = ' '
		} else if in[i] < 32 || in[i] > 126 {
			buf[i%16] = '.'
		} else {
			buf[i%16] = in[i]
		}
		if i%16 == 15 {
			out.WriteString(fmt.Sprintf("  %s\n", string(buf[:])))
		}
	}
	return out.String()
}
