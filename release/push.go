package release

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sollie/trackarr/utils/defaults"
	"strings"
	"time"

	"github.com/sollie/trackarr/config"
	"github.com/sollie/trackarr/database"
	"github.com/sollie/trackarr/database/models"
	"github.com/sollie/trackarr/utils/web"
	"github.com/sollie/trackarr/ws"

	"github.com/jpillora/backoff"
)

/* Structs */

type (
	pvrResponse struct {
		Approved   bool
		Rejections []string
	}
	pushRequest struct {
		Title            string `json:"title"`
		DownloadUrl      string `json:"downloadUrl"`
		Size             int64  `json:"size"`
		Indexer          string `json:"indexer"`
		DownloadProtocol string `json:"downloadProtocol"`
		Protocol         string `json:"protocol"`
		PublishDate      string `json:"publishDate"`
	}
)

/* Seams */

// PushResult is the PVR's verdict on a pushed release.
type PushResult struct {
	Approved   bool
	Rejections []string
}

// Pusher delivers a release to a PVR. The default adapter speaks the Sonarr/Radarr v3 api.
type Pusher interface {
	PushRelease(pvr *config.PvrConfig, request pushRequest) (*PushResult, error)
}

// Publisher records and broadcasts a pushed release. The default adapter writes to the
// database and the websocket releases topic.
type Publisher interface {
	Publish(info *config.ReleaseInfo, pvrName string, approved bool) error
}

type httpPusher struct{}

type dbPublisher struct{}

/* Public */

func (r *Release) Push(pvr *config.PvrConfig, delay *int64, torrentUrl *string) {
	if delay != nil && *delay > 0 {
		r.Log.Debugf("Delaying: %s (pvr: %s)", r.Info.TorrentName, pvr.Name)
		time.Sleep(time.Duration(*delay) * time.Second)
	}
	r.Log.Debugf("Pushing: %s (pvr: %s)", r.Info.TorrentName, pvr.Name)

	pvrRequest := pushRequest{
		Title:            r.Info.TorrentName,
		DownloadUrl:      *torrentUrl,
		Size:             0,
		Indexer:          r.Tracker.Info.LongName,
		DownloadProtocol: "torrent",
		Protocol:         "torrent",
		PublishDate:      r.Info.ReleaseTime,
	}

	if r.Info.SizeBytes > 0 {
		pvrRequest.Size = r.Info.SizeBytes
	}

	pusher := r.Pusher
	if pusher == nil {
		pusher = httpPusher{}
	}

	result, err := pusher.PushRelease(pvr, pvrRequest)
	if err != nil {
		r.Log.WithError(err).Errorf("Failed pushing: %s (pvr: %s)", r.Info.TorrentName, pvr.Name)
		return
	}

	r.Log.Infof("Pushed: %s (pvr: %s - approved: %v)", r.Info.TorrentName, pvr.Name, result.Approved)
	if len(result.Rejections) > 0 {
		r.Log.Tracef("Push rejected: %s (pvr: %s - reasons: %q)", r.Info.TorrentName, pvr.Name,
			strings.Join(result.Rejections, ", "))
	}

	publisher := r.Publisher
	if publisher == nil {
		publisher = dbPublisher{}
	}

	if err := publisher.Publish(r.Info, pvr.Name, result.Approved); err != nil {
		r.Log.WithError(err).Errorf("Failed saving release in database: %q", r.Info.TorrentName)
	}
}

/* Adapters */

func (httpPusher) PushRelease(pvr *config.PvrConfig, pvrRequest pushRequest) (*PushResult, error) {
	requestUrl := pvr.URL
	if !strings.Contains(pvr.URL, "/api/") {
		requestUrl = web.JoinURL(pvr.URL, "api/v3/release/push")
	}

	headers := web.Header{
		"X-Api-Key": pvr.ApiKey,
	}

	resp, err := web.Do(web.Request{Method: web.POST, URL: requestUrl,
		Timeout: defaults.GetOrDefaultInt(&pvr.Timeout, 30), Body: web.JSON{Value: pvrRequest}, Headers: headers, Retry: &web.Retry{
			MaxAttempts: 6,
			RetryableStatusCodes: []int{
				504,
			},
			Backoff: backoff.Backoff{
				Jitter: true,
				Min:    500 * time.Millisecond,
				Max:    10 * time.Second,
			},
		}})
	if err != nil {
		return nil, err
	}

	defer web.DrainAndClose(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected response: %q", resp.Status)
	}

	pvrResp := make([]pvrResponse, 0)
	if err := json.NewDecoder(resp.Body).Decode(&pvrResp); err != nil {
		return nil, errors.Wrap(err, "failed decoding push response")
	} else if len(pvrResp) < 1 {
		return nil, errors.New("empty push response")
	}

	return &PushResult{Approved: pvrResp[0].Approved, Rejections: pvrResp[0].Rejections}, nil
}

func (dbPublisher) Publish(info *config.ReleaseInfo, pvrName string, approved bool) error {
	dbRelease, err := models.NewPushedRelease(database.DB, info.TorrentName, info.TrackerName, pvrName, approved)
	if err != nil {
		return err
	}

	pushMessage := &ws.WebsocketMessage{
		Type: "release",
		Data: dbRelease,
	}

	if jsonData, err := pushMessage.ToJsonString(); err == nil {
		ws.BroadcastTopic("releases", jsonData)
	}

	return nil
}
