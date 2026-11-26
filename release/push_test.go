package release

import (
	"errors"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/sollie/trackarr/config"
)

type fakePusher struct {
	got    pushRequest
	result *PushResult
	err    error
}

func (f *fakePusher) PushRelease(_ *config.PvrConfig, r pushRequest) (*PushResult, error) {
	f.got = r
	return f.result, f.err
}

type fakePublisher struct {
	calls    int
	pvr      string
	approved bool
}

func (f *fakePublisher) Publish(_ *config.ReleaseInfo, pvr string, approved bool) error {
	f.calls++
	f.pvr, f.approved = pvr, approved
	return nil
}

func newTestRelease(p Pusher, pub Publisher) *Release {
	long := "Long Name"
	return &Release{
		Tracker:   &config.TrackerInstance{Info: &config.TrackerInfo{LongName: long}},
		Log:       logrus.NewEntry(logrus.New()),
		Info:      &config.ReleaseInfo{TorrentName: "rel", SizeBytes: 5, ReleaseTime: "t"},
		Pusher:    p,
		Publisher: pub,
	}
}

func TestPush(t *testing.T) {
	url := "http://dl"
	pvr := &config.PvrConfig{Name: "sonarr"}

	t.Run("publishes approved result", func(t *testing.T) {
		p := &fakePusher{result: &PushResult{Approved: true}}
		pub := &fakePublisher{}
		newTestRelease(p, pub).Push(pvr, nil, &url)

		if p.got.Title != "rel" || p.got.DownloadUrl != url || p.got.Size != 5 || p.got.Indexer != "Long Name" {
			t.Errorf("request = %+v", p.got)
		}
		if pub.calls != 1 || pub.pvr != "sonarr" || !pub.approved {
			t.Errorf("publisher = %+v", pub)
		}
	})

	t.Run("does not publish when push fails", func(t *testing.T) {
		pub := &fakePublisher{}
		newTestRelease(&fakePusher{err: errors.New("x")}, pub).Push(pvr, nil, &url)
		if pub.calls != 0 {
			t.Errorf("publisher called %d times", pub.calls)
		}
	})
}
