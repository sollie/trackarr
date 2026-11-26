package tracker

import (
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/sollie/trackarr/config"
	"github.com/sollie/trackarr/utils/web"
	"golang.org/x/time/rate"
)

/* Const */
const (
	btnApiUrl               = "https://api.broadcasthe.net/"
	btnTimeout              = 60
	btnApiRateLimitDuration = time.Hour
	btnApiRateLimit         = 150
)

/* Struct */
type Btn struct {
	client  *lookupClient
	tracker *config.TrackerInstance
	apiKey  string
	baseURL string
}

type (
	btnRequest struct {
		Id      int32     `json:"id"`
		JsonRPC string    `json:"jsonrpc"`
		Method  string    `json:"method"`
		Params  [2]string `json:"params"`
	}
)

/* Private */

func newBtn(tracker *config.TrackerInstance) (Interface, error) {
	log := log.WithField("api", tracker.Name)

	settings, err := requireSettings(tracker.Config.Settings, "api_key")
	if err != nil {
		return nil, err
	}

	return &Btn{
		client: &lookupClient{
			log: log,
			limiter: web.GetRateLimiter(tracker.Name, btnApiRateLimit, btnApiRateLimitDuration, btnApiRateLimit, func(rl *rate.Limiter) error {
				if !rl.Allow() {
					return errors.New("rate limit reached, too many requests to the host")
				}

				return nil
			}),
			timeout:    btnTimeout,
			attempts:   6,
			minBackoff: 2 * time.Second,
			maxBackoff: 6 * time.Second,
		},
		tracker: tracker,
		apiKey:  settings[0],
		baseURL: btnApiUrl,
	}, nil
}

/* Interface */

func (t *Btn) GetReleaseInfo(torrent *config.ReleaseInfo) (*TorrentInfo, error) {
	torrentUrl, err := url.Parse(torrent.TorrentURL)
	if err != nil {
		return nil, fmt.Errorf("malformed URL: %s", torrent.TorrentURL)
	}

	torrentId := torrentUrl.Query().Get("id")
	if torrentId == "" {
		return nil, fmt.Errorf("missing mandatory torrentId: %#v", torrent)
	}

	// the request is built per call, so concurrent lookups never share a mutable body
	body, err := t.client.fetch(lookupRequest{
		Method:  web.POST,
		URL:     t.baseURL,
		Headers: web.Header{"Content-Type": "application/json"},
		Body: web.JSON{Value: btnRequest{
			Id:      1,
			JsonRPC: "2.0",
			Method:  "getTorrentById",
			Params:  [2]string{t.apiKey, torrentId},
		}},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed retrieving torrent bytes from: %s", torrentId)
	}

	var btnInfo struct {
		Id     int32
		Result struct {
			TorrentID   string // "1289304",
			Size        string // "1942229405",
			ReleaseName string // "Shahs.of.Sunset.S08E09.iNTERNAL.1080p.WEB.h264-TRUMP",
		}
	}
	if err := t.client.decode(body, &btnInfo); err != nil {
		return nil, err
	}

	return &TorrentInfo{
		Name: btnInfo.Result.ReleaseName,
		Size: btnInfo.Result.Size,
	}, nil
}
