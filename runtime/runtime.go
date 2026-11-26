package runtime

import (
	"net/http"

	"github.com/sollie/trackarr/config"
	"github.com/sollie/trackarr/ircclient"
	"github.com/sollie/trackarr/loghook"
	"github.com/sollie/trackarr/tasks"
)

var (
	// State
	Loghook = loghook.NewLoghooker()
	Tracker = make(map[string]*config.TrackerInstance)
	Irc     = make(map[string]*ircclient.IRCClient)
	Web     *http.Server
	Tasks   *tasks.Tasks
)
