package apis

import (
	"net/http"

	"github.com/sollie/trackarr/config"
	"github.com/sollie/trackarr/logger"
	"github.com/sollie/trackarr/version"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
)

/* Structs */

type UpdateResponse struct {
	UpdateAvailable bool   `json:"update_available" xml:"update_available"`
	LatestVersion   string `json:"latest_version" xml:"latest_version"`
	CurrentVersion  string `json:"current_version" xml:"current_version"`
}

/* Public */

func UpdateStatus(c *echo.Context) error {
	// log
	log := logger.GetLogger("api").WithFields(logrus.Fields{"client": c.RealIP()})

	// is there an update available?
	usingLatest, latestVersion := version.Trackarr.IsLatest()
	log.Debugf("Latest version: %q, update available: %v", latestVersion, !usingLatest)

	// return response
	return c.JSON(http.StatusOK, &UpdateResponse{
		UpdateAvailable: !usingLatest,
		LatestVersion:   latestVersion.String(),
		CurrentVersion:  config.Build.Version,
	})
}
