package main

import (
	"encoding/json"
	"maps"
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
	theme := p.getConfiguration().customTheme
	if theme == "" {
		return
	}

	pref := model.Preference{
		UserId:   user.Id,
		Category: model.PreferenceCategoryTheme,
		Name:     "",
		Value:    theme,
	}

	appErr := p.API.UpdatePreferencesForUser(user.Id, []model.Preference{pref})
	if appErr != nil {
		p.API.LogError("Failed to set theme preference for user", "user_id", user.Id, "err", appErr.Error())
	}
}

// UserHasJoinedTeam is invoked after a user has joined a team. It sets the team's configured
// theme, unless the user has already chosen a theme of their own.
func (p *Plugin) UserHasJoinedTeam(c *plugin.Context, teamMember *model.TeamMember, actor *model.User) {
	config := p.getConfiguration()
	if len(config.teamThemes) == 0 {
		return
	}

	team, appErr := p.API.GetTeam(teamMember.TeamId)
	if appErr != nil {
		p.API.LogError("Failed to get team", "team_id", teamMember.TeamId, "err", appErr.Error())
		return
	}

	theme, ok := config.teamThemes[team.Name]
	if !ok {
		return
	}

	prefs, appErr := p.API.GetPreferencesForUser(teamMember.UserId)
	if appErr != nil {
		p.API.LogError("Failed to get preferences for user", "user_id", teamMember.UserId, "err", appErr.Error())
		return
	}

	for _, pref := range prefs {
		if pref.Category != model.PreferenceCategoryTheme {
			continue
		}
		if pref.Name == teamMember.TeamId {
			return
		}
		// A global theme set by this plugin doesn't count as the user's own choice.
		if pref.Name == "" && (config.customTheme == "" || !maps.Equal(themeColors(pref.Value), themeColors(config.customTheme))) {
			return
		}
	}

	pref := model.Preference{
		UserId:   teamMember.UserId,
		Category: model.PreferenceCategoryTheme,
		Name:     teamMember.TeamId,
		Value:    theme,
	}

	appErr = p.API.UpdatePreferencesForUser(teamMember.UserId, []model.Preference{pref})
	if appErr != nil {
		p.API.LogError("Failed to set team theme preference for user", "user_id", teamMember.UserId, "team_id", teamMember.TeamId, "err", appErr.Error())
	}
}

// themeColors returns a theme preference value as a map, without its type, normalized the way
// the server does when it saves a theme.
func themeColors(value string) map[string]string {
	pref := model.Preference{Category: model.PreferenceCategoryTheme, Value: value}
	pref.PreUpdate()

	var colors map[string]string
	_ = json.Unmarshal([]byte(pref.Value), &colors)
	delete(colors, "type")
	return colors
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
