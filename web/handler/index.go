package handler

import (
	"net/http"

	"github.com/sollie/trackarr/config"

	"github.com/labstack/echo/v5"
)

/* Public */

func Index(c *echo.Context) error {
	return c.Render(http.StatusOK, "index.html", map[string]any{
		"apikey":  config.Config.Server.ApiKey,
		"baseurl": config.Config.Server.BaseURL,
		"version": config.Build.Version,
	})
}
