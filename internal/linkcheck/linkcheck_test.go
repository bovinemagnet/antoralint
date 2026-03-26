package linkcheck

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCheck_OK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := New(2, 5*time.Second)
	results := c.Check([]string{ts.URL})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].IsOK() {
		t.Errorf("expected OK, got status %d err %v", results[0].StatusCode, results[0].Error)
	}
}

func TestCheck_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	c := New(2, 5*time.Second)
	results := c.Check([]string{ts.URL})
	if !results[0].IsDead() {
		t.Errorf("expected dead link, got status %d", results[0].StatusCode)
	}
}

func TestCheck_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := New(2, 5*time.Second)
	results := c.Check([]string{ts.URL})
	if !results[0].IsTransient() {
		t.Errorf("expected transient failure for 500, got status %d", results[0].StatusCode)
	}
}

func TestCheck_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := New(2, 100*time.Millisecond)
	results := c.Check([]string{ts.URL})
	if !results[0].IsTransient() {
		t.Errorf("expected transient (timeout), got status %d err %v", results[0].StatusCode, results[0].Error)
	}
}

func TestCheck_HEADFallbackToGET(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := New(2, 5*time.Second)
	results := c.Check([]string{ts.URL})
	if !results[0].IsOK() {
		t.Errorf("expected OK after HEAD->GET fallback, got status %d", results[0].StatusCode)
	}
}

func TestCheck_Concurrency(t *testing.T) {
	var count int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	urls := make([]string, 10)
	for i := range urls {
		urls[i] = ts.URL
	}

	c := New(3, 5*time.Second)
	results := c.Check(urls)
	if len(results) != 10 {
		t.Errorf("expected 10 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.IsOK() {
			t.Errorf("result[%d] not OK: status %d err %v", i, r.StatusCode, r.Error)
		}
	}
	if atomic.LoadInt64(&count) != 10 {
		t.Errorf("expected 10 requests, got %d", atomic.LoadInt64(&count))
	}
}
