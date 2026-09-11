package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errSleepCancelled = errors.New("sleep cancelled")

type throttleLog struct {
	sleeps  []time.Duration
	notices []time.Duration
}

// testThrottle returns a throttle that records requested sleeps and notifications instead of waiting or printing.
func testThrottle(now time.Time) (*throttle, *throttleLog) {
	log := &throttleLog{}
	t := newThrottle(http.DefaultTransport)
	t.now = func() time.Time { return now }
	t.sleep = func(_ context.Context, d time.Duration) error {
		log.sleeps = append(log.sleeps, d)
		return nil
	}
	t.notify = func(d time.Duration) { log.notices = append(log.notices, d) }
	return t, log
}

// get issues a GET through the round tripper and returns the final status code.
func get(t *testing.T, rt http.RoundTripper, url string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	discard(resp)
	return resp.StatusCode
}

func TestThrottle_RetriesOnRetryAfter(t *testing.T) {
	// Given
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set(headerRetryAfter, "7")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	throttle, log := testThrottle(time.Now())

	// When
	status := get(t, throttle, server.URL)

	// Then
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, int32(2), calls.Load())
	assert.Equal(t, []time.Duration{7 * time.Second}, log.sleeps)
	assert.Equal(t, []time.Duration{7 * time.Second}, log.notices)
}

func TestThrottle_RetriesOnPrimaryLimitReset(t *testing.T) {
	// Given
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	reset := now.Add(90 * time.Second)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set(headerRateLimitRemaining, "0")
			w.Header().Set(headerRateLimitReset, strconv.FormatInt(reset.Unix(), 10))
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	throttle, log := testThrottle(now)

	// When
	status := get(t, throttle, server.URL)

	// Then
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, int32(2), calls.Load())
	assert.Equal(t, []time.Duration{91 * time.Second}, log.sleeps)
}

func TestThrottle_DoesNotRetryPlainForbidden(t *testing.T) {
	// Given
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	throttle, log := testThrottle(time.Now())

	// When
	status := get(t, throttle, server.URL)

	// Then
	assert.Equal(t, http.StatusForbidden, status)
	assert.Equal(t, int32(1), calls.Load())
	assert.Empty(t, log.sleeps)
	assert.Empty(t, log.notices)
}

func TestThrottle_GivesUpAfterMaxAttempts(t *testing.T) {
	// Given
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set(headerRetryAfter, "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	throttle, log := testThrottle(time.Now())

	// When
	status := get(t, throttle, server.URL)

	// Then
	assert.Equal(t, http.StatusTooManyRequests, status)
	assert.Equal(t, int32(maxAttempts), calls.Load())
	assert.Len(t, log.sleeps, maxAttempts-1)
}

func TestThrottle_RefusesToWaitPastMaxBackoff(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerRetryAfter, strconv.Itoa(int(maxBackoff.Seconds())+1))
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	throttle, log := testThrottle(time.Now())
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	// When
	resp, err := throttle.RoundTrip(req) //nolint:bodyclose // resp is nil on the error path

	// Then
	require.ErrorIs(t, err, errBackoffTooLong)
	assert.Nil(t, resp)
	assert.Empty(t, log.sleeps)
}

func TestThrottle_StopsWhenContextCancelledDuringBackoff(t *testing.T) {
	// Given
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set(headerRetryAfter, "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	throttle, _ := testThrottle(time.Now())
	throttle.sleep = func(_ context.Context, _ time.Duration) error { return errSleepCancelled }
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	// When
	resp, err := throttle.RoundTrip(req) //nolint:bodyclose // resp is nil on the error path

	// Then
	require.ErrorIs(t, err, errSleepCancelled)
	assert.Nil(t, resp)
	assert.Equal(t, int32(1), calls.Load())
}

func TestRetryAfter(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	type test struct {
		name    string
		status  int
		headers map[string]string
		wait    time.Duration
		limited bool
	}
	tests := []test{
		{name: "ok", status: http.StatusOK, headers: map[string]string{headerRetryAfter: "5"}},
		{name: "not found", status: http.StatusNotFound},
		{name: "plain forbidden", status: http.StatusForbidden},
		{name: "bare too many requests", status: http.StatusTooManyRequests, wait: minBackoff, limited: true},
		{name: "forbidden with remaining budget", status: http.StatusForbidden, headers: map[string]string{headerRateLimitRemaining: "12"}},
		{name: "retry-after", status: http.StatusTooManyRequests, headers: map[string]string{headerRetryAfter: "30"}, wait: 30 * time.Second, limited: true},
		{name: "retry-after zero is clamped", status: http.StatusTooManyRequests, headers: map[string]string{headerRetryAfter: "0"}, wait: minBackoff, limited: true},
		{name: "reset in the future", status: http.StatusForbidden, headers: map[string]string{headerRateLimitRemaining: "0", headerRateLimitReset: "1000060"}, wait: 61 * time.Second, limited: true},
		{name: "reset in the past is clamped", status: http.StatusForbidden, headers: map[string]string{headerRateLimitRemaining: "0", headerRateLimitReset: "999000"}, wait: minBackoff, limited: true},
		{name: "reset unparseable", status: http.StatusForbidden, headers: map[string]string{headerRateLimitRemaining: "0", headerRateLimitReset: "soon"}, wait: minBackoff, limited: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tc.status, Header: http.Header{}}
			for k, v := range tc.headers {
				resp.Header.Set(k, v)
			}
			wait, limited := retryAfter(resp, now)
			assert.Equal(t, tc.limited, limited)
			assert.Equal(t, tc.wait, wait)
		})
	}
}
