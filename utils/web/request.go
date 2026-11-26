package web

import (
	"github.com/sollie/trackarr/config"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/sollie/trackarr/logger"
	"github.com/sollie/trackarr/utils/lists"

	"net/http"

	"github.com/imroc/req/v3"
	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
)

var (
	// Logging
	log = logger.GetLogger("web")
)

type (
	// Header holds request headers
	Header = map[string]string
	// Query holds url query parameters
	Query = map[string]string
	// Form is a url-encoded form request body
	Form = map[string]string
	// JSON is a json request body, marshalled from Value
	JSON struct{ Value interface{} }
)

/* Structs */

// HTTPMethod - The HTTP request method to use
type HTTPMethod int
type Retry struct {
	backoff.Backoff
	MaxAttempts          float64
	RetryableStatusCodes []int
	ExpectedContentType  string
}

const (
	// GET - Use GET HTTP method
	GET HTTPMethod = iota + 1
	// POST - Use POST HTTP method
	POST
)

// Request describes a single outgoing http request. Only Method and URL are mandatory.
type Request struct {
	Method  HTTPMethod
	URL     string
	Timeout int // seconds, 0 for none
	Headers Header
	Query   Query
	// Body is nil, a JSON, or a Form
	Body       interface{}
	Retry      *Retry
	Limiter    *RateLimiter
	NoRedirect bool
}

/* Private */

func userAgent() string {
	if config.Build != nil {
		return "trackarr/" + config.Build.Version
	}

	return "trackarr/v1.0.0"
}

/* Public */

// Do sends the request, honouring its rate limiter and retry policy. The caller's Headers are never modified.
// The caller must drain and close the response body.
func Do(r Request) (*http.Response, error) {
	if r.Method != GET && r.Method != POST {
		log.Error("Request method has not been implemented")
		return nil, errors.New("request method has not been implemented")
	}

	// prepare client (no shared cookies, body is read by the caller)
	client := req.C().
		SetCookieJar(nil).
		DisableAutoReadResponse().
		DisableKeepAlives().
		SetTimeout(time.Duration(r.Timeout) * time.Second).
		SetUserAgent(userAgent())
	if r.NoRedirect {
		client.SetRedirectPolicy(req.NoRedirectPolicy())
	}

	build := func() *req.Request {
		request := client.R().SetHeaders(r.Headers).SetQueryParams(r.Query)
		switch body := r.Body.(type) {
		case JSON:
			request.SetBodyJsonMarshal(body.Value)
		case *JSON:
			request.SetBodyJsonMarshal(body.Value)
		case Form:
			request.SetFormData(body)
		}

		return request
	}

	var retry Retry
	if r.Retry != nil {
		retry.MaxAttempts = r.Retry.MaxAttempts
		retry.RetryableStatusCodes = r.Retry.RetryableStatusCodes
		retry.ExpectedContentType = r.Retry.ExpectedContentType
		retry.Factor = r.Retry.Factor
		retry.Jitter = r.Retry.Jitter
		retry.Min = r.Retry.Min
		retry.Max = r.Retry.Max
	}

	var resp *http.Response
	var err error

	// Exponential back-off
	for {
		if r.Limiter != nil {
			if err := r.Limiter.Take(r.Limiter.limiter); err != nil {
				return nil, err
			}
		}

		var response *req.Response
		if r.Method == GET {
			response, err = build().Get(r.URL)
		} else {
			response, err = build().Post(r.URL)
		}
		resp = nil
		if response != nil {
			resp = response.Response
		}

		// validate response
		if err != nil {
			log.WithError(err).Debugf("Failed requesting: %q", r.URL)
			if os.IsTimeout(err) {
				if retry.MaxAttempts == 0 || retry.Attempt() >= retry.MaxAttempts {
					return nil, err
				}

				d := retry.Duration()
				log.Debugf("Retrying failed request in %s: %q", d, r.URL)
				time.Sleep(d)
				continue
			}

			return nil, err
		}

		log.Tracef("Request URL: %s", resp.Request.URL)
		log.Tracef("Request Response: %s", resp.Status)

		if retry.MaxAttempts == 0 || retry.Attempt() >= retry.MaxAttempts {
			break
		}

		// check status code vs retryable ones
		if lists.IntListContains(resp.StatusCode, retry.RetryableStatusCodes) {
			DrainAndClose(resp.Body)

			d := retry.Duration()
			log.Debugf("Retrying failed request in %s: %d - %q", d, resp.StatusCode, r.URL)

			time.Sleep(d)
			continue
		}

		// check response content type vs expected one
		if retry.ExpectedContentType != "" {
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), strings.ToLower(retry.ExpectedContentType)) {
				DrainAndClose(resp.Body)

				d := retry.Duration()
				log.Debugf("Retrying failed request in %s: %d %s - %q", d, resp.StatusCode, contentType, r.URL)

				time.Sleep(d)
				continue
			}
		}

		break
	}

	return resp, err
}

// BodyBytes sends the request and returns the full response body.
func BodyBytes(r Request) ([]byte, error) {
	resp, err := Do(r)
	if err != nil {
		return nil, err
	}
	defer DrainAndClose(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Errorf("Failed reading response body for url: %q", r.URL)
		return nil, errors.Wrap(err, "failed reading url response body")
	}

	return body, nil
}

// BodyString sends the request and returns the full response body as a string.
func BodyString(r Request) (string, error) {
	bodyBytes, err := BodyBytes(r)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}
