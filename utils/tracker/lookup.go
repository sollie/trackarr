package tracker

import (
	"time"

	"github.com/imroc/req"
	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/sollie/trackarr/utils/maps"
	"github.com/sollie/trackarr/utils/web"
)

// lookupClient owns the plumbing shared by every tracker api adapter: json requests with
// retry/backoff behind the tracker's rate limiter, and logged decoding of the response.
// Adapters only supply what varies: url, auth, request body and response mapping.
type lookupClient struct {
	log     *logrus.Entry
	limiter *web.RateLimiter
	timeout int
	// retry policy
	attempts   float64
	minBackoff time.Duration
	maxBackoff time.Duration
}

// lookupRequest is what an adapter supplies per lookup.
type lookupRequest struct {
	Method  web.HTTPMethod
	URL     string
	Headers req.Header
	Query   req.QueryParam
	Body    interface{}
}

// fetch performs the request and returns the raw json body.
func (c *lookupClient) fetch(r lookupRequest) ([]byte, error) {
	body, err := web.BodyBytes(web.Request{
		Method:  r.Method,
		URL:     r.URL,
		Timeout: c.timeout,
		Headers: r.Headers,
		Query:   r.Query,
		Body:    r.Body,
		Limiter: c.limiter,
		Retry: &web.Retry{
			MaxAttempts:         c.attempts,
			ExpectedContentType: "application/json",
			Backoff: backoff.Backoff{
				Jitter: true,
				Min:    c.minBackoff,
				Max:    c.maxBackoff,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	c.log.Tracef("Raw API Response : %s", string(body))
	return body, nil
}

// decode unmarshals a json response into v.
func (c *lookupClient) decode(body []byte, v interface{}) error {
	if err := json.Unmarshal(body, v); err != nil {
		c.log.WithError(err).Errorf("Failed unmarshalling response: %#v", string(body))
		return errors.Wrap(err, "failed unmarshalling response")
	}

	c.log.Tracef("GetReleaseInfo Response: %+v", v)
	return nil
}

// requireSettings returns the named tracker settings, or an error naming the first one missing.
func requireSettings(settings map[string]string, keys ...string) ([]string, error) {
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		value, err := maps.GetStringMapValue(settings, key, false)
		if err != nil {
			return nil, errors.WithMessage(err, key+" setting missing")
		}

		values = append(values, value)
	}

	return values, nil
}
