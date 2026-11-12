# Set Default Theme

This plugin is used to set the default theme for your users when they create their account.

### Install
1. Create a custom theme in your user settings: **Settings** -> **Display** -> **Theme**. You can find some example custom themes [here](https://avasconcelos114.github.io/mattermost-themes).
2. Copy the JSON representation for the theme you would like to make the default.
3. Navigate to the settings page for this plugin in the system console.
4. Paste the JSON string into the "Default Theme" config setting.

When any new account is created, the new user will have their theme set to this value.

### Team themes

You can also set a theme per team in the "Team Themes" setting. Paste a JSON object that maps team names to theme JSON. The team name is the part of the team URL after the site URL, e.g. `engineering` in `https://mattermost.example.com/engineering`:

```json
{
  "engineering": {"sidebarBg": "#001e27", "...": "..."},
  "sales": {"sidebarBg": "#ffffff", "...": "..."}
}
```

When a user joins one of these teams, the team's theme is applied for that team. Users who have already chosen a theme of their own, either globally or for that team, keep it. Users can switch back to a single theme for all teams by changing their theme and checking **Apply new theme to all my teams**.

![image](https://user-images.githubusercontent.com/6913320/112346908-cb634700-8c9c-11eb-9a6a-621a2893c528.png)
