package format

import "fmt"

const msInS = 1000
const msInM = msInS * 60
const msInH = msInM * 60

// Humanize returns unit milliseconds in a simple human-readable form
func Humanize(ms uint) string {
	switch {
	case ms < msInS:
		return fmt.Sprintf("%dms", ms)
	case ms < msInM:
		return fmt.Sprintf("%ds %dms", ms/msInS, ms%msInS)
	case ms < msInH:
		return fmt.Sprintf("%dm %ds", ms/msInM, (ms%msInM)/msInS)
	default:
		return fmt.Sprintf("%dh %dm", ms/msInH, (ms%msInH)/msInM)
	}
}
