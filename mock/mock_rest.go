package mock

import (
	"io"
	"net/http"

	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

// RestMock is used for mocking RESTClient in go-gh
type RestMock struct {
	mock.Mock
}

// Do is a mock implementation of RESTClient.Do
func (m *RestMock) Do(method, path string, body io.Reader, response interface{}) error {
	args := m.Called(method, path, body, response)
	return args.Error(0) //nolint:wrapcheck
}

// DoWithContext is a mock implementation of RESTClient.DoWithContext
func (m *RestMock) DoWithContext(ctx context.Context, method, path string, body io.Reader, response interface{}) error {
	args := m.Called(ctx, method, path, body, response)
	return args.Error(0) //nolint:wrapcheck
}

// Delete is a mock implementation of RESTClient.Delete
func (m *RestMock) Delete(path string, response interface{}) error {
	args := m.Called(path, response)
	return args.Error(0) //nolint:wrapcheck
}

// Get is a mock implementation of RESTClient.Get
func (m *RestMock) Get(path string, response interface{}) error {
	args := m.Called(path, response)
	return args.Error(0) //nolint:wrapcheck
}

// Patch is a mock implementation of RESTClient.Patch
func (m *RestMock) Patch(path string, body io.Reader, response interface{}) error {
	args := m.Called(path, body, response)
	return args.Error(0) //nolint:wrapcheck
}

// Post is a mock implementation of RESTClient.Post
func (m *RestMock) Post(path string, body io.Reader, response interface{}) error {
	args := m.Called(path, body, response)
	return args.Error(0) //nolint:wrapcheck
}

// Put is a mock implementation of RESTClient.Put
func (m *RestMock) Put(path string, body io.Reader, response interface{}) error {
	args := m.Called(path, body, response)
	return args.Error(0) //nolint:wrapcheck
}

// Request is a mock implementation of RESTClient.Request
func (m *RestMock) Request(method, path string, body io.Reader) (*http.Response, error) {
	args := m.Called(method, path, body)
	return nil, args.Error(0) //nolint:wrapcheck
}

// RequestWithContext is a mock implementation of RESTClient.RequestWithContext
func (m *RestMock) RequestWithContext(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	args := m.Called(ctx, method, path, body)
	return nil, args.Error(0) //nolint:wrapcheck
}
