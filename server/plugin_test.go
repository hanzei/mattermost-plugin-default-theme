package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/mock"
)

// newTestPlugin returns a plugin with the given configuration and a mock API whose
// expectations are asserted when the test ends.
func newTestPlugin(t *testing.T, config *configuration) (*Plugin, *plugintest.API) {
	testAPI := &plugintest.API{}
	t.Cleanup(func() { testAPI.AssertExpectations(t) })
	plugin := &Plugin{}
	plugin.SetAPI(testAPI)
	plugin.setConfiguration(config)
	return plugin, testAPI
}

func TestUserHasBeenCreatedValidTheme(t *testing.T) {
	theme := `{"sidebarBg":"#001e27","type":"custom"}`

	plugin, testAPI := newTestPlugin(t, &configuration{
		customTheme: theme,
	})

	testAPI.On("UpdatePreferencesForUser", "userid", []model.Preference{{
		UserId:   "userid",
		Category: "theme",
		Name:     "",
		Value:    theme,
	}}).Once().Return(nil)
	plugin.UserHasBeenCreated(nil, &model.User{Id: "userid"})
}

func TestUserHasBeenCreatedBlankTheme(t *testing.T) {
	plugin, _ := newTestPlugin(t, &configuration{})

	plugin.UserHasBeenCreated(nil, &model.User{Id: "userid"})
}

func TestUserHasBeenCreatedErrorSettingPreferences(t *testing.T) {
	plugin, testAPI := newTestPlugin(t, &configuration{
		customTheme: `{"sidebarBg":"#001e27","type":"custom"}`,
	})

	testAPI.On("UpdatePreferencesForUser", "userid", mock.AnythingOfType("[]model.Preference")).Once().Return(&model.AppError{Message: "preference error"})
	testAPI.On("LogError", "Failed to set theme preference for user", "user_id", "userid", "err", "preference error").Once()
	plugin.UserHasBeenCreated(nil, &model.User{Id: "userid"})
}

func TestUserHasJoinedTeam(t *testing.T) {
	const (
		// global is CustomTheme as prepared by OnConfigurationChange. The server stores it as
		// storedGlobal, and stored it as storedOldGlobal before the plugin added the type.
		global          = `{"buttonBg":"rgba(0,0,0,0.5)","sidebarBg":"#FFF","type":"custom"}`
		storedGlobal    = `{"buttonBg":"#ffffff","sidebarBg":"#FFF","type":"custom"}`
		storedOldGlobal = `{"buttonBg":"#ffffff","sidebarBg":"#FFF"}`
		teamTheme       = `{"sidebarBg":"#001e27","type":"custom"}`
		ownTheme        = `{"buttonBg":"#000000","sidebarBg":"#FFF","type":"custom"}`
		userID          = "userid"
		teamID          = "teamid"
	)

	themePref := func(name, value string) model.Preference {
		return model.Preference{UserId: userID, Category: model.PreferenceCategoryTheme, Name: name, Value: value}
	}

	for name, tc := range map[string]struct {
		noGlobal bool
		prefs    []model.Preference
		apply    bool
	}{
		"no preferences": {
			apply: true,
		},
		"unrelated preferences only": {
			prefs: []model.Preference{{UserId: userID, Category: "display_settings", Name: "", Value: "x"}},
			apply: true,
		},
		"global theme set by plugin": {
			prefs: []model.Preference{themePref("", storedGlobal)},
			apply: true,
		},
		"global theme set by an older plugin version": {
			prefs: []model.Preference{themePref("", storedOldGlobal)},
			apply: true,
		},
		"theme for another team": {
			prefs: []model.Preference{themePref("otherteamid", ownTheme)},
			apply: true,
		},
		"own global theme": {
			prefs: []model.Preference{themePref("", ownTheme)},
		},
		"global theme when plugin has none configured": {
			noGlobal: true,
			prefs:    []model.Preference{themePref("", storedGlobal)},
		},
		"existing theme for this team": {
			prefs: []model.Preference{themePref("", storedGlobal), themePref(teamID, ownTheme)},
		},
	} {
		t.Run(name, func(t *testing.T) {
			config := &configuration{
				customTheme: global,
				teamThemes:  map[string]string{"engineering": teamTheme},
			}
			if tc.noGlobal {
				config.customTheme = ""
			}
			plugin, testAPI := newTestPlugin(t, config)

			testAPI.On("GetTeam", teamID).Once().Return(&model.Team{Id: teamID, Name: "engineering"}, nil)
			testAPI.On("GetPreferencesForUser", userID).Once().Return(tc.prefs, nil)
			if tc.apply {
				testAPI.On("UpdatePreferencesForUser", userID, []model.Preference{themePref(teamID, teamTheme)}).Once().Return(nil)
			}

			plugin.UserHasJoinedTeam(nil, &model.TeamMember{TeamId: teamID, UserId: userID}, nil)
		})
	}
}

func TestUserHasJoinedTeamNoTeamThemes(t *testing.T) {
	plugin, _ := newTestPlugin(t, &configuration{})

	plugin.UserHasJoinedTeam(nil, &model.TeamMember{TeamId: "teamid", UserId: "userid"}, nil)
}

func TestUserHasJoinedTeamNotConfigured(t *testing.T) {
	plugin, testAPI := newTestPlugin(t, &configuration{
		teamThemes: map[string]string{"engineering": `{"type":"custom"}`},
	})

	testAPI.On("GetTeam", "teamid").Once().Return(&model.Team{Id: "teamid", Name: "sales"}, nil)
	plugin.UserHasJoinedTeam(nil, &model.TeamMember{TeamId: "teamid", UserId: "userid"}, nil)
}

func TestUserHasJoinedTeamErrors(t *testing.T) {
	config := &configuration{
		teamThemes: map[string]string{"engineering": `{"type":"custom"}`},
	}
	member := &model.TeamMember{TeamId: "teamid", UserId: "userid"}
	team := &model.Team{Id: "teamid", Name: "engineering"}

	t.Run("get team", func(t *testing.T) {
		plugin, testAPI := newTestPlugin(t, config)
		testAPI.On("GetTeam", "teamid").Once().Return(nil, &model.AppError{Message: "team error"})
		testAPI.On("LogError", "Failed to get team", "team_id", "teamid", "err", "team error").Once()
		plugin.UserHasJoinedTeam(nil, member, nil)
	})

	t.Run("get preferences", func(t *testing.T) {
		plugin, testAPI := newTestPlugin(t, config)
		testAPI.On("GetTeam", "teamid").Once().Return(team, nil)
		testAPI.On("GetPreferencesForUser", "userid").Once().Return(nil, &model.AppError{Message: "preference error"})
		testAPI.On("LogError", "Failed to get preferences for user", "user_id", "userid", "err", "preference error").Once()
		plugin.UserHasJoinedTeam(nil, member, nil)
	})

	t.Run("update preferences", func(t *testing.T) {
		plugin, testAPI := newTestPlugin(t, config)
		testAPI.On("GetTeam", "teamid").Once().Return(team, nil)
		testAPI.On("GetPreferencesForUser", "userid").Once().Return(nil, nil)
		testAPI.On("UpdatePreferencesForUser", "userid", mock.AnythingOfType("[]model.Preference")).Once().Return(&model.AppError{Message: "preference error"})
		testAPI.On("LogError", "Failed to set team theme preference for user", "user_id", "userid", "team_id", "teamid", "err", "preference error").Once()
		plugin.UserHasJoinedTeam(nil, member, nil)
	})
}
