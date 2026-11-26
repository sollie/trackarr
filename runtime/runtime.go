package runtime

import (
	"github.com/sollie/trackarr/config"
	"github.com/sollie/trackarr/ircclient"
	"github.com/sollie/trackarr/loghook"
	"github.com/sollie/trackarr/tasks"

	"github.com/labstack/echo/v4"
)

var (
	// State
	Loghook = loghook.NewLoghooker()
	Tracker = make(map[string]*config.TrackerInstance)
	Irc     = make(map[string]*ircclient.IRCClient)
	Web     *echo.Echo
	Tasks   *tasks.Tasks
)
