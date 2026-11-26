package web

import (
	"github.com/sollie/trackarr/config"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/sollie/trackarr/logger"
	"github.com/sollie/trackarr/utils/lists"

	"github.com/imroc/req"
	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
)

var (
	// Logging
	log        = logger.GetLogger("web")
	httpClient = *req.Client()
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
	Headers req.Header
	Query   req.QueryParam
	// Body is any body accepted by req (req.BodyJSON, req.Param, string, []byte, ...)
	Body       interface{}
	Retry      *Retry
	Limiter    *RateLimiter
	NoRedirect bool
}

/* Private */

func init() {
	// dont json escape html
	req.SetJSONEscapeHTML(false)

	// use timeout from Do
	httpClient.Timeout = time.Duration(0)
}

func userAgent() string {
	if config.Build != nil {
		return "trackarr/" + config.Build.Version
	}

	return "trackarr/v1.0.0"
}

/* Public */

// Do sends the request, honouring its rate limiter and retry policy. The caller's Headers are never modified.
func Do(r Request) (*req.Resp, error) {
	if r.Method != GET && r.Method != POST {
		log.Error("Request method has not been implemented")
		return nil, errors.New("request method has not been implemented")
	}

	// prepare client
	client := httpClient
	if r.Timeout > 0 {
		client.Timeout = time.Duration(r.Timeout) * time.Second
	}
	if r.NoRedirect {
		noRedirect(&client)
	}

	// prepare request inputs
	headers := req.Header{}
	for k, v := range r.Headers {
		headers[k] = v
	}
	headers["User-Agent"] = userAgent()

	inputs := []interface{}{&client, headers}
	if r.Query != nil {
		inputs = append(inputs, r.Query)
	}
	if r.Body != nil {
		inputs = append(inputs, r.Body)
	}

	var retry Retry
	if r.Retry != nil {
		retry = *r.Retry
	}

	var resp *req.Resp
	var err error

	// Exponential back-off
	for {
		if r.Limiter != nil {
			if err := r.Limiter.Take(r.Limiter.limiter); err != nil {
				return nil, err
			}
		}

		if r.Method == GET {
			resp, err = req.Get(r.URL, inputs...)
		} else {
			resp, err = req.Post(r.URL, inputs...)
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

		log.Tracef("Request URL: %s", resp.Request().URL)
		log.Tracef("Request Response: %s", resp.Response().Status)

		if retry.MaxAttempts == 0 || retry.Attempt() >= retry.MaxAttempts {
			break
		}

		// check status code vs retryable ones
		if lists.IntListContains(resp.Response().StatusCode, retry.RetryableStatusCodes) {
			DrainAndClose(resp.Response().Body)

			d := retry.Duration()
			log.Debugf("Retrying failed request in %s: %d - %q", d, resp.Response().StatusCode, r.URL)

			time.Sleep(d)
			continue
		}

		// check response content type vs expected one
		if retry.ExpectedContentType != "" {
			contentType := resp.Response().Header.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), strings.ToLower(retry.ExpectedContentType)) {
				DrainAndClose(resp.Response().Body)

				d := retry.Duration()
				log.Debugf("Retrying failed request in %s: %d %s - %q", d, resp.Response().StatusCode, contentType, r.URL)

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
	defer DrainAndClose(resp.Response().Body)

	body, err := ioutil.ReadAll(resp.Response().Body)
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
