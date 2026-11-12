package main

import (
	"encoding/json"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// UserHasBeenCreated is invoked after a user was created.
func (p *Plugin) UserHasBeenCreated(c *plugin.Context, user *model.User) {
	theme := p.getConfiguration().CustomTheme
	if theme == "" {
		return
	}

	m := map[string]any{}
	err := json.Unmarshal([]byte(theme), &m)
	if err != nil {
		p.API.LogError("Failed to parse theme during user creation", "err", err.Error())
		return
	}

	pref := model.Preference{
		UserId:   user.Id,
		Category: model.PreferenceCategoryTheme,
		Name:     "",
		Value:    theme,
	}
	prefs := []model.Preference{pref}

	appErr := p.API.UpdatePreferencesForUser(user.Id, prefs)
	if appErr != nil {
		p.API.LogError("Failed to set theme preference for user", "user_id", user.Id, "err", appErr.Error())
	}
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
