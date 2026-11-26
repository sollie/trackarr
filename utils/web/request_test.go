package web

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/imroc/req"
	"github.com/jpillora/backoff"
)

func TestDoBasicAndUserAgent(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	headers := req.Header{"X-Test": "1"}
	body, err := BodyString(Request{Method: GET, URL: srv.URL, Headers: headers})
	if err != nil || body != "ok" {
		t.Fatalf("body = %q, err = %v", body, err)
	}
	if ua == "" {
		t.Error("user agent not set")
	}
	if _, ok := headers["User-Agent"]; ok {
		t.Error("caller headers were mutated")
	}
}

func TestDoRetriesStatusCode(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := Do(Request{Method: GET, URL: srv.URL, Retry: &Retry{
		MaxAttempts:          5,
		RetryableStatusCodes: []int{504},
		Backoff:              backoff.Backoff{Min: time.Millisecond, Max: time.Millisecond},
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer DrainAndClose(resp.Response().Body)

	if resp.Response().StatusCode != 200 || calls != 3 {
		t.Errorf("status = %d, calls = %d", resp.Response().StatusCode, calls)
	}
}

func TestDoRetriesUnexpectedContentType(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 2 {
			w.Header().Set("Content-Type", "text/html")
		} else {
			w.Header().Set("Content-Type", "application/JSON; charset=utf-8")
		}
	}))
	defer srv.Close()

	resp, err := Do(Request{Method: GET, URL: srv.URL, Retry: &Retry{
		MaxAttempts:         5,
		ExpectedContentType: "application/json",
		Backoff:             backoff.Backoff{Min: time.Millisecond, Max: time.Millisecond},
	}})
	if err != nil {
		t.Fatal(err)
	}
	DrainAndClose(resp.Response().Body)

	if calls != 2 {
		t.Errorf("calls = %d", calls)
	}
}

func TestDoNoRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	for noRedirect, want := range map[bool]int{true: 302, false: 200} {
		resp, err := Do(Request{Method: GET, URL: srv.URL, NoRedirect: noRedirect})
		if err != nil {
			t.Fatal(err)
		}
		DrainAndClose(resp.Response().Body)
		if resp.Response().StatusCode != want {
			t.Errorf("NoRedirect=%v: status = %d, want %d", noRedirect, resp.Response().StatusCode, want)
		}
	}
}

func TestDoRejectsUnsupportedMethod(t *testing.T) {
	if _, err := Do(Request{Method: HTTPMethod(99), URL: "http://localhost"}); err == nil {
		t.Fatal("expected error")
	}
}
