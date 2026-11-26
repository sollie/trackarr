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
}
