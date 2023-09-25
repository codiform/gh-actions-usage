package format

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFormatters(t *testing.T) {
	type test struct {
		name         string
		expectedType interface{}
	}
	tests := []test{
		{name: "human", expectedType: humanFormatter{}},
		{name: "tsv", expectedType: tsvFormatter{}},
		{name: "yaml"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			formatter, err := GetFormatter(tc.name)
			if tc.expectedType == nil {
				assert.Errorf(t, err, "unknown formatter %s", tc.name)
			} else {
				assert.IsType(t, tc.expectedType, formatter)
			}
		})
	}
}
