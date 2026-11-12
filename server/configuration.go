package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/mattermost/mattermost/server/public/model"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type configuration struct {
	CustomTheme string
	TeamThemes  string

	// customTheme and teamThemes hold the themes as they are saved as preferences, computed in
	// OnConfigurationChange. teamThemes is keyed by team name.
	customTheme string
	teamThemes  map[string]string
}

// Clone deep copies the configuration.
func (c *configuration) Clone() *configuration {
	clone := *c
	clone.teamThemes = maps.Clone(c.teamThemes)
	return &clone
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made. Invalid
// themes are skipped and reported in the returned error, which the server logs, so that a mistake
// in one theme doesn't disable the others.
func (p *Plugin) OnConfigurationChange() error {
	configuration := new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return fmt.Errorf("failed to load plugin configuration: %w", err)
	}

	var errs []error

	if configuration.CustomTheme != "" {
		theme, err := prepareTheme([]byte(configuration.CustomTheme))
		if err != nil {
			errs = append(errs, fmt.Errorf("error parsing default theme while saving config: %w", err))
		}
		configuration.customTheme = theme
	}

	if configuration.TeamThemes != "" {
		rawThemes := map[string]json.RawMessage{}
		if err := json.Unmarshal([]byte(configuration.TeamThemes), &rawThemes); err != nil {
			errs = append(errs, fmt.Errorf("error parsing team themes while saving config: %w", err))
			rawThemes = nil
		}

		configuration.teamThemes = make(map[string]string, len(rawThemes))
		for _, teamName := range slices.Sorted(maps.Keys(rawThemes)) {
			if !model.IsValidTeamName(teamName) {
				errs = append(errs, fmt.Errorf("error parsing team themes while saving config: %q is not a valid team name; use the lowercase name from the team URL", teamName))
				continue
			}
			theme, err := prepareTheme(rawThemes[teamName])
			if err != nil {
				errs = append(errs, fmt.Errorf("error parsing theme for team %q while saving config: %w", teamName, err))
				continue
			}
			configuration.teamThemes[teamName] = theme
		}
	}

	p.setConfiguration(configuration)

	return errors.Join(errs...)
}

// prepareTheme checks that data is a JSON object with string values, as the server requires for
// theme preferences. If it has no "type", it's set to "custom", which the webapp needs to show
// it as a custom theme.
func prepareTheme(data []byte) (string, error) {
	var theme map[string]string
	if err := json.Unmarshal(data, &theme); err != nil {
		return "", err
	}
	if theme == nil {
		return "", errors.New("theme must be a JSON object")
	}

	if theme["type"] == "" {
		theme["type"] = "custom"
	}

	return model.MapToJSON(theme), nil
}
