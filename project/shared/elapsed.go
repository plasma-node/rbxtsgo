package shared

import (
	"fmt"
	"time"
)

// FormatElapsed is a build's duration for people to read: whole milliseconds under a second, then seconds.
func FormatElapsed(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%d ms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2f s", d.Seconds())
}
