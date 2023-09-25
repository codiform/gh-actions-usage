package format

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHumanize(t *testing.T) {
	type test struct {
		name      string
		humanized string
		ms        uint
	}
	tests := []test{
		{name: "test<s", ms: 321, humanized: "321ms"},
		{name: "s<test<m", ms: 12_345, humanized: "12s 345ms"},
		{name: "m<test<h", ms: 754_567, humanized: "12m 34s"},
		{name: "h<test", ms: 45_240_000, humanized: "12h 34m"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.humanized, Humanize(tc.ms))
		})
	}
}
