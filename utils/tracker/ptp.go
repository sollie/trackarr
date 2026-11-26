package tracker

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sollie/trackarr/config"
	"github.com/sollie/trackarr/utils/web"
	"golang.org/x/time/rate"
)

/* Const */
const (
	ptpTorrentUrl           = "https://passthepopcorn.me/torrents.php"
	ptpTimeout              = 30
	ptpApiRateLimitDuration = time.Second
	ptpApiRateLimit         = 1
)

/* Struct */
type Ptp struct {
	client  *lookupClient
	tracker *config.TrackerInstance
	headers web.Header
	baseURL string
}

/* Private */

func newPtp(tracker *config.TrackerInstance) (Interface, error) {
	log := log.WithField("api", tracker.Name)

	settings, err := requireSettings(tracker.Config.Settings, "api_user", "api_key")
	if err != nil {
		return nil, err
	}

	return &Ptp{
		client: &lookupClient{
			log: log,
			limiter: web.GetRateLimiter(tracker.Name, ptpApiRateLimit, ptpApiRateLimitDuration, 1, func(rl *rate.Limiter) error {
				return rl.Wait(context.Background())
			}),
			timeout:    ptpTimeout,
			attempts:   5,
			minBackoff: 2 * time.Second,
			maxBackoff: 6 * time.Second,
		},
		tracker: tracker,
		headers: web.Header{
			"ApiUser": settings[0],
			"ApiKey":  settings[1],
		},
		baseURL: ptpTorrentUrl,
	}, nil
}

/* Interface */

func (t *Ptp) GetReleaseInfo(torrent *config.ReleaseInfo) (*TorrentInfo, error) {
	if torrent.TorrentId == "" {
		return nil, fmt.Errorf("missing mandatory torrentId: %#v", torrent)
	}

	body, err := t.client.fetch(lookupRequest{
		Method:  web.GET,
		URL:     t.baseURL,
		Headers: t.headers,
		Query:   web.Query{"torrentid": torrent.TorrentId},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed retrieving torrent info bytes for: %s", torrent.TorrentId)
	}

	var ptpInfo struct {
		Torrents []struct {
			Id          string // "1368",
			Size        string // "1468434432",
			ReleaseName string // "The Godfather Part 3",
		}
	}
	if err := t.client.decode(body, &ptpInfo); err != nil {
		return nil, err
	}

	for _, v := range ptpInfo.Torrents {
		if v.Id == torrent.TorrentId {
			return &TorrentInfo{
				Name:     v.ReleaseName,
				Category: "",
				Size:     v.Size,
			}, nil
		}
	}

	return nil, fmt.Errorf("no release found with id: %s", torrent.TorrentId)
}
