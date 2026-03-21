package format

import "fmt"

const msInS = 1000
const msInM = msInS * 60
const msInH = msInM * 60

// Humanize returns unit milliseconds in a simple human-readable form
func Humanize(ms uint) string {
	hours := ms / msInH
	minutes := (ms % msInH) / msInM
	seconds := (ms % msInM) / msInS
	millis := ms % msInS

	if hours > 0 {
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		if seconds == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	if seconds > 0 {
		if millis == 0 {
			return fmt.Sprintf("%ds", seconds)
		}
		return fmt.Sprintf("%ds %dms", seconds, millis)
	}
	return fmt.Sprintf("%dms", millis)
}
