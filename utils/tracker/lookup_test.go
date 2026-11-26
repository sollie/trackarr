package tracker

import (
	stdjson "encoding/json"
	"github.com/sollie/trackarr/utils/web"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/sollie/trackarr/config"
)

func testClient() *lookupClient {
	return &lookupClient{
		log:        logrus.NewEntry(logrus.New()),
		timeout:    5,
		attempts:   2,
		minBackoff: time.Millisecond,
		maxBackoff: time.Millisecond,
	}
}

func jsonServer(t *testing.T, check func(r *http.Request), body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		check(r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
}

func TestPtpGetReleaseInfo(t *testing.T) {
	srv := jsonServer(t, func(r *http.Request) {
		if r.Header.Get("ApiUser") != "u" || r.Header.Get("ApiKey") != "k" {
			t.Errorf("unexpected request: %v %v", r.Header, r.URL)
		}
	}, `{"Torrents":[{"Id":"1","Size":"1","ReleaseName":"no"},{"Id":"2","Size":"2048","ReleaseName":"Movie"}]}`)
	defer srv.Close()

	a := &Ptp{client: testClient(), baseURL: srv.URL, headers: web.Header{"ApiUser": "u", "ApiKey": "k"}}
	info, err := a.GetReleaseInfo(&config.ReleaseInfo{TorrentId: "2"})
	if err != nil || info.Name != "Movie" || info.Size != "2048" {
		t.Fatalf("info = %+v, err = %v", info, err)
	}

	if _, err := a.GetReleaseInfo(&config.ReleaseInfo{TorrentId: "9"}); err == nil {
		t.Error("expected not-found error")
	}
	if _, err := a.GetReleaseInfo(&config.ReleaseInfo{}); err == nil {
		t.Error("expected missing id error")
	}
}

func TestBtnGetReleaseInfo(t *testing.T) {
	srv := jsonServer(t, func(r *http.Request) {
		var got btnRequest
		if err := stdjson.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		if r.Method != http.MethodPost || got.Params != [2]string{"key", "77"} || got.Method != "getTorrentById" {
			t.Errorf("unexpected request: %v %+v", r.Method, got)
		}
	}, `{"id":1,"result":{"TorrentID":"77","Size":"1942","ReleaseName":"Show.S01"}}`)
	defer srv.Close()

	a := &Btn{client: testClient(), baseURL: srv.URL, apiKey: "key"}
	info, err := a.GetReleaseInfo(&config.ReleaseInfo{TorrentURL: "https://btn/torrents.php?id=77"})
	if err != nil || info.Name != "Show.S01" || info.Size != "1942" {
		t.Fatalf("info = %+v, err = %v", info, err)
	}

	if _, err := a.GetReleaseInfo(&config.ReleaseInfo{TorrentURL: "https://btn/torrents.php"}); err == nil {
		t.Error("expected missing id error")
	}
}

func TestMtvGetReleaseInfo(t *testing.T) {
	srv := jsonServer(t, func(r *http.Request) {
		if r.URL.Query().Get("action") != "torrent" || r.URL.Query().Get("id") != "5" {
			t.Errorf("unexpected query: %v", r.URL.RawQuery)
		}
	}, `{"status":"success","response":{"group":{"name":"Film","categoryName":"Movies"},"torrent":{"id":5,"size":999}}}`)
	defer srv.Close()

	expiry := time.Now().Add(time.Hour)
	a := &Mtv{client: testClient(), torrentURL: srv.URL, cookieExpiry: &expiry}
	info, err := a.GetReleaseInfo(&config.ReleaseInfo{TorrentURL: "https://mtv/torrents.php?id=5"})
	if err != nil || info.Name != "Film" || info.Category != "Movies" || info.Size != "999" {
		t.Fatalf("info = %+v, err = %v", info, err)
	}
}

func TestRequireSettings(t *testing.T) {
	if got, err := requireSettings(map[string]string{"a": "1", "b": "2"}, "a", "b"); err != nil || got[1] != "2" {
		t.Errorf("got %v, err %v", got, err)
	}
	if _, err := requireSettings(map[string]string{"a": "1"}, "a", "b"); err == nil {
		t.Error("expected error for missing setting")
	}
}
