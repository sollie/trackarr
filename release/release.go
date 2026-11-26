package release

import (
	"github.com/sollie/trackarr/config"

	"github.com/sirupsen/logrus"
)

/* Structs */

type Release struct {
	Tracker *config.TrackerInstance
	Log     *logrus.Entry
	Info    *config.ReleaseInfo

	// optional adapters, defaulting to the PVR http api and database/websocket publisher
	Pusher    Pusher
	Publisher Publisher
}
