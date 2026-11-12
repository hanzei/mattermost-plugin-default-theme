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
		config.TeamThemes = `{"engineering":{"sidebarBg":"#001e27"}}`
	}).Once().Return(nil)

	err := plugin.OnConfigurationChange()
	require.EqualError(t, err, "error parsing default theme while saving config: unexpected end of JSON input")

	// The invalid default theme is skipped; team themes still apply.
	config := plugin.getConfiguration()
	require.Empty(t, config.customTheme)
	require.Equal(t, map[string]string{"engineering": `{"sidebarBg":"#001e27","type":"custom"}`}, config.teamThemes)
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
		wantErrs   []string
	}{
		"blank": {},
		"valid": {
			teamThemes: `{"engineering":{"sidebarBg":"#001e27"},"sales":{"sidebarBg":"#ffffff","type":"Onyx"}}`,
			want: map[string]string{
				"engineering": `{"sidebarBg":"#001e27","type":"custom"}`,
				"sales":       `{"sidebarBg":"#ffffff","type":"Onyx"}`,
			},
		},
		"invalid entries are skipped": {
			teamThemes: `{"engineering":{"sidebarBg":"#001e27"},"Sales":{"sidebarBg":"#ffffff"},"support":{"sidebarBg":"#001e27","opacity":0.5}}`,
			want:       map[string]string{"engineering": `{"sidebarBg":"#001e27","type":"custom"}`},
			wantErrs: []string{
				`error parsing team themes while saving config: "Sales" is not a valid team name; use the lowercase name from the team URL`,
				`error parsing theme for team "support" while saving config: json: cannot unmarshal number`,
			},
		},
		"invalid JSON": {
			teamThemes: `{"engineering":`,
			want:       map[string]string{},
			wantErrs:   []string{"error parsing team themes while saving config: unexpected end of JSON input"},
		},
		"not an object": {
			teamThemes: `["engineering"]`,
			want:       map[string]string{},
			wantErrs:   []string{"error parsing team themes while saving config: json: cannot unmarshal array"},
		},
		"theme not an object": {
			teamThemes: `{"engineering":"#001e27"}`,
			want:       map[string]string{},
			wantErrs:   []string{`error parsing theme for team "engineering" while saving config: json: cannot unmarshal string`},
		},
		"null theme": {
			teamThemes: `{"engineering":null}`,
			want:       map[string]string{},
			wantErrs:   []string{`error parsing theme for team "engineering" while saving config: theme must be a JSON object`},
		},
	} {
		t.Run(name, func(t *testing.T) {
			testAPI := &plugintest.API{}
			plugin := Plugin{}
			plugin.SetAPI(testAPI)

			testAPI.On("LoadPluginConfiguration", mock.AnythingOfType("*main.configuration")).Once().Run(func(args mock.Arguments) {
				config := args.Get(0).(*configuration)
				config.CustomTheme = `{"sidebarBg":"#ffffff"}`
				config.TeamThemes = tc.teamThemes
			}).Return(nil)

			err := plugin.OnConfigurationChange()
			if len(tc.wantErrs) == 0 {
				require.NoError(t, err)
			}
			for _, wantErr := range tc.wantErrs {
				require.ErrorContains(t, err, wantErr)
			}

			config := plugin.getConfiguration()
			require.Equal(t, tc.want, config.teamThemes)
			// Invalid team themes don't affect the default theme.
			require.Equal(t, `{"sidebarBg":"#ffffff","type":"custom"}`, config.customTheme)
		})
	}
}
