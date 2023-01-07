package format

import "fmt"

const ms_in_s = 1000
const ms_in_m = ms_in_s * 60
const ms_in_h = ms_in_m * 60

func Humanize(ms uint) string {
	switch {
	case ms < ms_in_s:
		return fmt.Sprintf("%dms", ms)
	case ms < ms_in_m:
		return fmt.Sprintf("%ds %dms", ms/ms_in_s, ms%ms_in_s)
	case ms < ms_in_h:
		return fmt.Sprintf("%dm %ds", ms/ms_in_m, (ms%ms_in_m)/ms_in_s)
	default:
		return fmt.Sprintf("%dh %dm", ms/ms_in_h, (ms%ms_in_h)/ms_in_m)
	}
}
