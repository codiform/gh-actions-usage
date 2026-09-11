// Package mock provides test mock implementations for the gh-actions-usage client interfaces.
package mock

import (
	"github.com/stretchr/testify/mock"
)

// RestMock is a testify mock implementation of client.REST
type RestMock struct {
	mock.Mock
}

// Get is a mock implementation of REST.Get
func (m *RestMock) Get(path string, response any) error {
	args := m.Called(path, response)
	return args.Error(0) //nolint:wrapcheck
}
