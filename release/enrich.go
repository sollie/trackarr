package release

import (
	"github.com/docker/go-units"
	"github.com/imroc/req"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/sollie/trackarr/cache"
	"github.com/sollie/trackarr/config"
	stringutils "github.com/sollie/trackarr/utils/strings"
	"github.com/sollie/trackarr/utils/torrent"
	"github.com/sollie/trackarr/utils/tracker"
)

/* Const */
const TorrentFileTimeout = 30

/* Structs */

// EnrichSource describes where the missing release information came from.
type EnrichSource int

const (
	// SourceNone - nothing was looked up (no api, bencode disabled)
	SourceNone EnrichSource = iota
	// SourceAPI - the tracker api supplied the information
	SourceAPI
	// SourceBencode - the torrent file supplied the information
	SourceBencode
)

// Enrichment is the outcome of Enrich.
type Enrichment struct {
	Source EnrichSource
	// AddedToCache is true when the downloaded torrent file was stored in the cache.
	AddedToCache bool
}

// enricher holds the seams the enrichment policy depends on.
type enricher struct {
	tracker *config.TrackerInstance
	info    *config.ReleaseInfo
	log     *logrus.Entry

	getAPI       func(*config.TrackerInstance) (tracker.Interface, error)
	fetchTorrent func(url string, timeout int, headers req.Header) (*torrent.Data, error)
	addToCache   func(key string, item *cache.CacheItem)
}

/* Public */

// Enrich fills in missing release information (name, category, size, files).
//
// Policy: the tracker api is tried first, then the torrent file (bencode) if
// enabled. If nothing usable can be obtained and bencode is disabled, an error
// is returned and the release must be dropped.
func (r *Release) Enrich() (Enrichment, error) {
	return (&enricher{
		tracker:      r.Tracker,
		info:         r.Info,
		log:          r.Log,
		getAPI:       tracker.GetApi,
		fetchTorrent: torrent.GetTorrentDetails,
		addToCache: func(key string, item *cache.CacheItem) {
			go cache.AddItem(key, item)
		},
	}).run()
}

/* Private */

func (e *enricher) bencodeEnabled() bool {
	return e.tracker.Config.Bencode.Name || e.tracker.Config.Bencode.Size
}

func (e *enricher) run() (Enrichment, error) {
	usedApi, err := e.apiLookup()
	if err != nil {
		return Enrichment{}, err
	}

	e.convertSize()

	if usedApi {
		return Enrichment{Source: SourceAPI}, nil
	}

	usedBencode, err := e.bencodeLookup()
	if err != nil {
		e.log.WithError(err).Error("Failed performing bencode lookup")
		return Enrichment{}, err
	}
	if usedBencode {
		return Enrichment{Source: SourceBencode, AddedToCache: true}, nil
	}

	return Enrichment{Source: SourceNone}, nil
}

// convertSize converts the parsed release size string to bytes (required by the pvr).
func (e *enricher) convertSize() {
	if e.info.SizeString == "" {
		return
	}

	byteSize, err := units.FromHumanSize(e.info.SizeString)
	if err != nil {
		e.log.WithError(err).Warnf("Failed converting parsed release size %q to bytes", e.info.SizeString)
		e.info.SizeBytes = 0
		return
	}

	e.info.SizeBytes = byteSize
	e.log.Tracef("Converted parsed release size %q to %d bytes", e.info.SizeString, e.info.SizeBytes)
}

// apiFallback decides what to do when the api could not supply the information.
func (e *enricher) apiFallback() (bool, error) {
	if !e.bencodeEnabled() {
		e.log.Warnf("Aborting push of release as bencode disabled for torrent: %q", e.info.TorrentName)
		return false, errors.New("api lookup failed and bencode disabled")
	}

	return false, nil
}

func (e *enricher) apiLookup() (bool, error) {
	trackerApi, _ := e.getAPI(e.tracker)
	if trackerApi == nil {
		// api not implemented for this tracker
		return false, nil
	}

	torrentInfo, err := trackerApi.GetReleaseInfo(e.info)
	if err != nil {
		e.log.WithError(err).Errorf("Failed looking up missing info via api for torrent: %q", e.info.TorrentName)
		return e.apiFallback()
	}
	if torrentInfo == nil {
		return e.apiFallback()
	}

	e.log.Debugf("Retrieved torrent info via api: %+v", torrentInfo)

	e.info.TorrentName = stringutils.NewOrExisting(&torrentInfo.Name, e.info.TorrentName)
	e.info.Category = stringutils.NewOrExisting(&torrentInfo.Category, e.info.Category)
	e.info.SizeString = stringutils.NewOrExisting(&torrentInfo.Size, e.info.SizeString)

	if e.info.SizeString == "" && !e.bencodeEnabled() {
		e.log.Warnf("Aborting push of release as api response was incomplete and bencode disabled"+
			" for torrent: %q", e.info.TorrentName)
		return false, errors.New("api lookup response incomplete and bencode disabled")
	}

	return true, nil
}

func (e *enricher) bencodeLookup() (bool, error) {
	if !e.bencodeEnabled() {
		return false, nil
	}

	headers := req.Header{}
	if cookie, ok := e.tracker.Config.Settings["cookie"]; ok {
		headers["Cookie"] = cookie
	}

	torrentData, err := e.fetchTorrent(e.info.TorrentURL, TorrentFileTimeout, headers)
	if err != nil {
		return false, errors.WithMessage(err, "failed decoding details from torrent file")
	}

	e.info.Files = torrentData.Files

	if e.tracker.Config.Bencode.Name {
		e.info.TorrentName = torrentData.Name
	}

	if e.tracker.Config.Bencode.Size || e.info.SizeBytes == 0 {
		e.info.SizeBytes = torrentData.Size
	}

	e.addToCache(e.info.TorrentURL, &cache.CacheItem{
		Name:    e.info.TorrentName,
		Data:    torrentData.Bytes,
		Release: e.info,
	})

	return true, nil
}
