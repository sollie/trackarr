package release

import (
	"errors"
	"testing"

	"github.com/imroc/req"
	"github.com/sirupsen/logrus"

	"github.com/sollie/trackarr/cache"
	"github.com/sollie/trackarr/config"
	"github.com/sollie/trackarr/utils/torrent"
	"github.com/sollie/trackarr/utils/tracker"
)

type fakeAPI struct {
	info *tracker.TorrentInfo
	err  error
}

func (f fakeAPI) GetReleaseInfo(*config.ReleaseInfo) (*tracker.TorrentInfo, error) {
	return f.info, f.err
}

func newEnricher(api tracker.Interface, bencodeName, bencodeSize bool, td *torrent.Data, fetchErr error) (*enricher, *config.ReleaseInfo, *int) {
	info := &config.ReleaseInfo{TorrentName: "orig", TorrentURL: "http://x/t"}
	cached := 0
	return &enricher{
		tracker: &config.TrackerInstance{Config: &config.TrackerConfig{
			Bencode:  &config.TrackerBencodeConfig{Name: bencodeName, Size: bencodeSize},
			Settings: map[string]string{},
		}},
		info: info,
		log:  logrus.NewEntry(logrus.New()),
		getAPI: func(*config.TrackerInstance) (tracker.Interface, error) {
			if api == nil {
				return nil, errors.New("not implemented")
			}
			return api, nil
		},
		fetchTorrent: func(string, int, req.Header) (*torrent.Data, error) { return td, fetchErr },
		addToCache:   func(string, *cache.CacheItem) { cached++ },
	}, info, &cached
}

func TestEnrich(t *testing.T) {
	td := &torrent.Data{Name: "bn", Size: 42, Files: []string{"a"}}
	apiOK := fakeAPI{info: &tracker.TorrentInfo{Name: "api", Size: "1 KB"}}

	tests := []struct {
		name         string
		api          tracker.Interface
		bName, bSize bool
		wantSource   EnrichSource
		wantErr      bool
		wantName     string
		wantBytes    int64
		wantCached   int
	}{
		{"api success", apiOK, false, false, SourceAPI, false, "api", 1000, 0},
		{"api error, bencode disabled aborts", fakeAPI{err: errors.New("x")}, false, false, SourceNone, true, "orig", 0, 0},
		{"api nil info, bencode fallback", fakeAPI{}, true, true, SourceBencode, false, "bn", 42, 1},
		{"api incomplete, bencode disabled aborts", fakeAPI{info: &tracker.TorrentInfo{Name: "api"}}, false, false, SourceNone, true, "api", 0, 0},
		{"no api, bencode disabled", nil, false, false, SourceNone, false, "orig", 0, 0},
		{"no api, bencode name only keeps size", nil, true, false, SourceBencode, false, "bn", 42, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e, info, cached := newEnricher(tc.api, tc.bName, tc.bSize, td, nil)
			got, err := e.run()
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if got.Source != tc.wantSource || got.AddedToCache != (tc.wantCached > 0) {
				t.Errorf("enrichment = %+v", got)
			}
			if info.TorrentName != tc.wantName || info.SizeBytes != tc.wantBytes || *cached != tc.wantCached {
				t.Errorf("info = %+v cached = %d", info, *cached)
			}
		})
	}
}

func TestEnrichBencodeFetchError(t *testing.T) {
	e, _, _ := newEnricher(nil, true, true, nil, errors.New("boom"))
	if _, err := e.run(); err == nil {
		t.Fatal("expected error")
	}
}
