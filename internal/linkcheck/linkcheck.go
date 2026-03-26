package linkcheck

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Result represents the outcome of checking a single URL.
type Result struct {
	URL        string
	StatusCode int
	Error      error
	TimedOut   bool
}

// IsOK returns true if the URL was successfully reached with a 2xx status.
func (r *Result) IsOK() bool {
	return r.Error == nil && r.StatusCode >= 200 && r.StatusCode < 400
}

// IsDead returns true if the URL returned a definite dead-link status (4xx).
func (r *Result) IsDead() bool {
	return r.Error == nil && r.StatusCode >= 400 && r.StatusCode < 500
}

// IsTransient returns true if the failure may be temporary (5xx, timeout, network error).
func (r *Result) IsTransient() bool {
	if r.TimedOut {
		return true
	}
	if r.Error != nil {
		return true
	}
	return r.StatusCode >= 500
}

// Checker validates external URLs.
type Checker struct {
	client      *http.Client
	concurrency int
}

// New creates a new Checker with the given concurrency limit and timeout per request.
func New(concurrency int, timeout time.Duration) *Checker {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Checker{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		concurrency: concurrency,
	}
}

// Check validates a list of URLs and returns results for each.
func (c *Checker) Check(urls []string) []*Result {
	results := make([]*Result, len(urls))
	sem := make(chan struct{}, c.concurrency)
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, u string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = c.checkOne(u)
		}(i, url)
	}

	wg.Wait()
	return results
}

func (c *Checker) checkOne(url string) *Result {
	result := &Result{URL: url}

	// Try HEAD first (timeout handled by http.Client.Timeout)
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		result.Error = err
		return result
	}
	req.Header.Set("User-Agent", "adoclint/0.1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		if isTimeout(err) {
			result.TimedOut = true
		}
		result.Error = err
		return result
	}
	resp.Body.Close()

	// If HEAD returns 405 Method Not Allowed, fall back to GET
	if resp.StatusCode == http.StatusMethodNotAllowed {
		req2, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			result.Error = err
			return result
		}
		req2.Header.Set("User-Agent", "adoclint/0.1.0")

		resp2, err := c.client.Do(req2)
		if err != nil {
			if isTimeout(err) {
				result.TimedOut = true
			}
			result.Error = err
			return result
		}
		resp2.Body.Close()
		result.StatusCode = resp2.StatusCode
		return result
	}

	result.StatusCode = resp.StatusCode
	return result
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	// net/http wraps timeout errors; check for the interface
	type timeouter interface {
		Timeout() bool
	}
	if t, ok := err.(timeouter); ok {
		return t.Timeout()
	}
	return false
}
