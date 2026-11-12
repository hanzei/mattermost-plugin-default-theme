package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOnConfigurationChangeValidTheme(t *testing.T) {
	theme := `{"awayIndicator":"#eae679","buttonBg":"#00aa55","buttonColor":"#041418","centerChannelBg":"#011010","centerChannelColor":"#1fbfec","codeTheme":"solarized-dark","dndIndicator":"#b79fa7","errorTextColor":"#dd2c45","linkColor":"#00aa55","mentionBg":"#00c463","mentionBj":"#00c463","mentionColor":"#001e27","mentionHighlightBg":"#032727","mentionHighlightLink":"#146c86","newMessageSeparator":"#146c86","onlineIndicator":"#0be3d6","sidebarBg":"#001e27","sidebarHeaderBg":"#001e27","sidebarHeaderTextColor":"#1fbfec","sidebarTeamBarBg":"#00181f","sidebarText":"#8ae9ff","sidebarTextActiveBorder":"#00c463","sidebarTextActiveColor":"#00c463","sidebarTextHoverBg":"#1a5c70","sidebarUnreadText":"#1fbfec"}`

	testAPI := &plugintest.API{}
	plugin := Plugin{}
	plugin.SetAPI(testAPI)

	testAPI.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Once().Run(func(args mock.Arguments) {
		config := args.Get(0).(*configuration)
		config.CustomTheme = theme
	}).Return(nil)

	err := plugin.OnConfigurationChange()
	require.Nil(t, err)

	config := plugin.getConfiguration()
	require.Equal(t, config.CustomTheme, theme)
	prepared := model.MapFromJSON(strings.NewReader(config.customTheme))
	require.Equal(t, "custom", prepared["type"])
}

func TestOnConfigurationChangeBlankTheme(t *testing.T) {
	theme := ""

	testAPI := &plugintest.API{}
	plugin := Plugin{}
	plugin.SetAPI(testAPI)

	testAPI.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Once().Run(func(args mock.Arguments) {
		config := args.Get(0).(*configuration)
		config.CustomTheme = theme
	}).Return(nil)

	err := plugin.OnConfigurationChange()
	require.Nil(t, err)

	config := plugin.getConfiguration()
	require.Equal(t, config.CustomTheme, theme)
}

func TestOnConfigurationChangeInvalidTheme(t *testing.T) {
	theme := `{"invalid": "json"`

	testAPI := &plugintest.API{}
	plugin := Plugin{}
	plugin.SetAPI(testAPI)

	testAPI.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Run(func(args mock.Arguments) {
		config := args.Get(0).(*configuration)
		config.CustomTheme = theme
	}).Once().Return(nil)

	err := plugin.OnConfigurationChange()
	require.NotNil(t, err)
	require.Equal(t, err.Error(), "error parsing default theme while saving config: unexpected end of JSON input")

	config := plugin.getConfiguration()
	require.Equal(t, config.CustomTheme, "")
}

func TestOnConfigurationChangeErrorLoadingConfig(t *testing.T) {
	testAPI := &plugintest.API{}
	plugin := Plugin{}
	plugin.SetAPI(testAPI)

	testAPI.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Once().Return(errors.New("some error"))

	err := plugin.OnConfigurationChange()
	require.NotNil(t, err)
	require.Equal(t, err.Error(), "failed to load plugin configuration: some error")

	config := plugin.getConfiguration()
	require.Equal(t, config.CustomTheme, "")
}

func TestOnConfigurationChangeTeamThemes(t *testing.T) {
	for name, tc := range map[string]struct {
		teamThemes string
		want       map[string]string
		wantErr    string
	}{
		"blank": {},
		"valid": {
			teamThemes: `{"engineering":{"sidebarBg":"#001e27"},"sales":{"sidebarBg":"#ffffff","type":"Onyx"}}`,
			want: map[string]string{
				"engineering": `{"sidebarBg":"#001e27","type":"custom"}`,
				"sales":       `{"sidebarBg":"#ffffff","type":"Onyx"}`,
			},
		},
		"invalid team name": {
			teamThemes: `{"Engineering":{"sidebarBg":"#001e27"}}`,
			wantErr:    `error parsing team themes while saving config: "Engineering" is not a valid team name; use the lowercase name from the team URL`,
		},
		"non-string theme value": {
			teamThemes: `{"engineering":{"sidebarBg":"#001e27","opacity":0.5}}`,
			wantErr:    `error parsing theme for team "engineering" while saving config: json: cannot unmarshal number`,
		},
		"invalid JSON": {
			teamThemes: `{"engineering":`,
			wantErr:    "error parsing team themes while saving config: unexpected end of JSON input",
		},
		"not an object": {
			teamThemes: `["engineering"]`,
			wantErr:    "error parsing team themes while saving config: json: cannot unmarshal array",
		},
		"theme not an object": {
			teamThemes: `{"engineering":"#001e27"}`,
			wantErr:    `error parsing theme for team "engineering" while saving config: json: cannot unmarshal string`,
		},
		"null theme": {
			teamThemes: `{"engineering":null}`,
			wantErr:    `error parsing theme for team "engineering" while saving config: theme must be a JSON object`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			testAPI := &plugintest.API{}
			plugin := Plugin{}
			plugin.SetAPI(testAPI)

			testAPI.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Once().Run(func(args mock.Arguments) {
				args.Get(0).(*configuration).TeamThemes = tc.teamThemes
			}).Return(nil)

			err := plugin.OnConfigurationChange()
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Nil(t, plugin.getConfiguration().teamThemes)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, plugin.getConfiguration().teamThemes)
		})
	}
}
