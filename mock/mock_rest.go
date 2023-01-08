package mock

import (
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
	"io"
	"net/http"
)

type RestMock struct {
	mock.Mock
}

func (m *RestMock) Do(method string, path string, body io.Reader, response interface{}) error {
	args := m.Called(method, path, body, response)
	return args.Error(0)
}

func (m *RestMock) DoWithContext(ctx context.Context, method string, path string, body io.Reader, response interface{}) error {
	args := m.Called(ctx, method, path, body, response)
	return args.Error(0)
}

func (m *RestMock) Delete(path string, response interface{}) error {
	args := m.Called(path, response)
	return args.Error(0)
}

func (m *RestMock) Get(path string, response interface{}) error {
	args := m.Called(path, response)
	return args.Error(0)
}

func (m *RestMock) Patch(path string, body io.Reader, response interface{}) error {
	args := m.Called(path, body, response)
	return args.Error(0)
}

func (m *RestMock) Post(path string, body io.Reader, response interface{}) error {
	args := m.Called(path, body, response)
	return args.Error(0)
}

func (m *RestMock) Put(path string, body io.Reader, response interface{}) error {
	args := m.Called(path, body, response)
	return args.Error(0)
}

func (m *RestMock) Request(method string, path string, body io.Reader) (*http.Response, error) {
	args := m.Called(method, path, body)
	return nil, args.Error(0)
}

func (m *RestMock) RequestWithContext(ctx context.Context, method string, path string, body io.Reader) (*http.Response, error) {
	args := m.Called(ctx, method, path, body)
	return nil, args.Error(0)
}
