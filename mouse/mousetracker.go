// NOTES: Not working if u using wayland, X11 is the way to go with this mouse position tool
package mousetrace

import (
	"fmt"
	"time"

	"github.com/go-vgo/robotgo"
)

func TrackEM() {
	var lastX, lastY int

	for {
		x, y := robotgo.Location()

		// Only print if the coordinates change
		if x != lastX || y != lastY {
			fmt.Printf("X: %d | Y: %d\n", x, y)
			lastX = x
			lastY = y
		}

		// Add a short delay to prevent capturing unnoticeably quick updates and avoid 100% CPU usage
		time.Sleep(100 * time.Millisecond)
	}
}
