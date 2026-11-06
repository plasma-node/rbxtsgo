package shared

import (
	"testing"
	"time"
)

func TestFormatElapsed(t *testing.T) {
	cases := map[time.Duration]string{
		0:                       "0 ms",
		87 * time.Millisecond:   "87 ms",
		999 * time.Millisecond:  "999 ms",
		time.Second:             "1.00 s",
		1234 * time.Millisecond: "1.23 s",
	}
	for input, want := range cases {
		if got := FormatElapsed(input); got != want {
			t.Errorf("FormatElapsed(%v) = %q, want %q", input, got, want)
		}
	}
}
