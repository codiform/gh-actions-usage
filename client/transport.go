package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

const (
	// requestsPerSecond keeps the extension under GitHub's secondary limit of roughly 900 REST points per minute.
	requestsPerSecond = 10
	// requestBurst is the number of requests that may be issued back-to-back before the limiter starts pacing.
	requestBurst = 10
	// maxAttempts is the total number of times a request will be tried when GitHub reports it as rate limited.
	maxAttempts = 3
	// minBackoff is the shortest pause before retrying a rate-limited request.
	minBackoff = time.Second
	// maxBackoff caps the pause before retrying, even if GitHub asks for longer.
	maxBackoff = 10 * time.Minute

	headerRetryAfter         = "Retry-After"
	headerRateLimitRemaining = "X-Ratelimit-Remaining"
	headerRateLimitReset     = "X-Ratelimit-Reset"
)

// errBackoffTooLong is returned when GitHub asks the client to wait longer than maxBackoff before retrying.
var errBackoffTooLong = errors.New("rate limit reset is too far away to wait for")

// throttle is an http.RoundTripper that paces requests to stay within GitHub's rate limits and
// retries requests that GitHub rejects as rate limited.
type throttle struct {
	next    http.RoundTripper
	limiter *rate.Limiter
	now     func() time.Time
	sleep   func(ctx context.Context, d time.Duration) error
	// notify is called before each backoff so the user can see why the command has paused.
	notify func(wait time.Duration)
}

func newThrottle(next http.RoundTripper) *throttle {
	return &throttle{
		next:    next,
		limiter: rate.NewLimiter(requestsPerSecond, requestBurst),
		now:     time.Now,
		sleep:   sleepContext,
		notify:  notifyStderr,
	}
}

func notifyStderr(wait time.Duration) {
	_, _ = fmt.Fprintf(os.Stderr, "GitHub rate limit reached; waiting %s before retrying...\n", wait.Round(time.Second))
}

// RoundTrip implements http.RoundTripper
func (t *throttle) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	for attempt := 1; ; attempt++ {
		err := t.limiter.Wait(ctx)
		if err != nil {
			return nil, fmt.Errorf("waiting for rate limiter: %w", err)
		}
		resp, err := t.next.RoundTrip(req)
		if err != nil {
			return nil, fmt.Errorf("sending request: %w", err)
		}
		wait, limited := retryAfter(resp, t.now())
		if !limited || attempt >= maxAttempts {
			return resp, nil
		}
		discard(resp)
		if wait > maxBackoff {
			return nil, fmt.Errorf("%w (%s)", errBackoffTooLong, wait.Round(time.Second))
		}
		t.notify(wait)
		err = t.sleep(ctx, wait)
		if err != nil {
			return nil, fmt.Errorf("waiting to retry rate-limited request: %w", err)
		}
	}
}

// retryAfter reports whether the response is a rate-limit rejection and, if so, how long to wait before retrying.
// GitHub signals rate limiting with 403 or 429 plus either a Retry-After header (secondary limits) or
// X-RateLimit-Remaining: 0 with an X-RateLimit-Reset timestamp (primary limit). A 429 is a rate-limit response
// even without those headers and gets the minimum backoff; a 403 without them is an ordinary authorization
// failure and is not retried.
func retryAfter(resp *http.Response, now time.Time) (time.Duration, bool) {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return 0, false
	}
	secs, err := strconv.Atoi(resp.Header.Get(headerRetryAfter))
	if err == nil {
		return clampBackoff(time.Duration(secs) * time.Second), true
	}
	if resp.Header.Get(headerRateLimitRemaining) == "0" {
		reset, err := strconv.ParseInt(resp.Header.Get(headerRateLimitReset), 10, 64)
		if err != nil {
			return minBackoff, true
		}
		return clampBackoff(time.Unix(reset, 0).Sub(now) + time.Second), true
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return minBackoff, true
	}
	return 0, false
}

func clampBackoff(d time.Duration) time.Duration {
	if d < minBackoff {
		return minBackoff
	}
	return d
}

func discard(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck
	}
}
