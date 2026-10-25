package main

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"spellcast"
	"spellcast/internal/claude"
	"spellcast/internal/config"
	"spellcast/internal/usage"
)

// notifier shows desktop notifications for events the user must see
// (for example "no API key").
type notifier struct {
	svc *notifications.NotificationService
}

func newNotifier() *notifier {
	return &notifier{svc: notifications.New()}
}

// Notify shows a notification. Failures are logged, never fatal.
func (n *notifier) Notify(id, title, body string) {
	err := n.svc.SendNotification(notifications.NotificationOptions{ID: id, Title: title, Body: body})
	if err != nil {
		slog.Warn("notification failed", "id", id, "title", title, "err", err)
	}
}

// NotifyNoKey tells the user that requests cannot work yet.
func (n *notifier) NotifyNoKey(err error) {
	if errors.Is(err, claude.ErrNoAPIKey) {
		n.Notify("no-api-key", "Spellcast: no API key", "Use the tray menu: Set API key…")
		return
	}
	n.Notify("no-api-key", "Spellcast: cannot read the API key", err.Error())
}

// tray owns the tray icon and its menu.
type tray struct {
	app          *application.App
	icon         *application.SystemTray
	onShowLast   func()
	onSetAPIKey  func()
	onOpenConfig func()
	onReload     func()
	onOpenLog    func()
	onAbout      func()
	onRemove     func()

	// mu guards cfg and the usage texts, and serializes menu rebuilds: tray
	// clicks, the usage tracker and the key window call in from different goroutines.
	mu  sync.Mutex
	cfg *config.Config

	// hasKey reports whether requests can work (a key from the credential
	// store or ANTHROPIC_API_KEY). Without one, "Set API key…" comes first.
	hasKey func() bool

	// Running totals, shown as read-only entries once there are any.
	usageToday, usageMonth string

	// Start with the OS: autostart reports the current state, onAutostart changes it.
	autostart   func() bool
	onAutostart func(enabled bool)
}

func newTray(app *application.App) *tray {
	t := &tray{app: app, icon: app.SystemTray.New()}
	t.icon.SetIcon(spellcast.TrayIcon)
	t.icon.SetTooltip("Spellcast")
	t.icon.OnClick(t.icon.OpenMenu) // a left-click opens the same menu as a right-click
	return t
}

// menuItem is one entry of the tray menu, independent of Wails (unit-tested).
type menuItem struct {
	Label     string
	ID        string // what a click does, see tray.handler
	Bold      bool   // Wails menus have no bold entries; the order alone marks it
	Disabled  bool
	Checkbox  bool
	Checked   bool
	Separator bool
	Sub       []menuItem
}

// actionLine is one configured hotkey for the read-only overview.
type actionLine struct{ Key, Name string }

// menuState is everything the menu depends on.
type menuState struct {
	Actions                []actionLine
	HasKey                 bool
	UsageToday, UsageMonth string
	Autostart              *bool // nil if the platform cannot start with the OS
}

var separator = menuItem{Separator: true}

// buildMenu returns the tray menu for s.
func buildMenu(s menuState) []menuItem {
	var m []menuItem
	if !s.HasKey {
		m = append(m, menuItem{Label: "Set API key…", ID: "set-key-top", Bold: true}, separator)
	}
	if len(s.Actions) == 0 {
		m = append(m, menuItem{Label: "No actions configured", Disabled: true})
	}
	for _, a := range s.Actions {
		m = append(m, menuItem{Label: fmt.Sprintf("CapsLock+%s   %s", a.Key, a.Name), Disabled: true})
	}
	m = append(m, separator, menuItem{Label: "Show last answer", ID: "show-last"})
	if s.UsageToday != "" {
		m = append(m, menuItem{Label: "Usage", Sub: []menuItem{
			{Label: s.UsageToday, Disabled: true},
			{Label: s.UsageMonth, Disabled: true},
		}})
	}
	settings := []menuItem{
		{Label: "Set API key…", ID: "set-key"},
		{Label: "Open config", ID: "open-config"},
		{Label: "Reload config", ID: "reload"},
	}
	if s.Autostart != nil {
		settings = append(settings, menuItem{Label: "Start with Windows", ID: "autostart", Checkbox: true, Checked: *s.Autostart})
	}
	return append(m, separator,
		menuItem{Label: "Settings", Sub: settings},
		menuItem{Label: "Help", Sub: []menuItem{
			{Label: "Open log", ID: "open-log"},
			{Label: "About Spellcast", ID: "about"},
			{Label: "Remove from Windows…", ID: "remove"},
		}},
		separator,
		menuItem{Label: "Quit", ID: "quit"},
	)
}

// SetConfig rebuilds the menu with the actions of cfg.
func (t *tray) SetConfig(cfg *config.Config) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cfg = cfg
	t.rebuildLocked()
}

// Refresh rebuilds the menu, for example after the API key changed.
func (t *tray) Refresh() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rebuildLocked()
}

// rebuildLocked builds the menu from the current state. t.mu must be held.
func (t *tray) rebuildLocked() {
	if t.cfg == nil {
		return
	}
	s := menuState{HasKey: t.hasKey == nil || t.hasKey(), UsageToday: t.usageToday, UsageMonth: t.usageMonth}
	for _, k := range t.cfg.ActionKeys() {
		a, _ := t.cfg.Action(k)
		s.Actions = append(s.Actions, actionLine{Key: k, Name: a.Name})
	}
	if t.autostart != nil {
		on := t.autostart()
		s.Autostart = &on
	}
	menu := t.app.NewMenu()
	t.render(menu, buildMenu(s))
	t.icon.SetMenu(menu)
}

func (t *tray) render(menu *application.Menu, items []menuItem) {
	for _, it := range items {
		switch {
		case it.Separator:
			menu.AddSeparator()
		case it.Sub != nil:
			t.render(menu.AddSubmenu(it.Label), it.Sub)
		case it.Checkbox:
			menu.AddCheckbox(it.Label, it.Checked).OnClick(func(ctx *application.Context) {
				slog.Info("tray menu", "entry", it.ID, "enabled", ctx.IsChecked())
				if t.onAutostart != nil {
					t.onAutostart(ctx.IsChecked())
				}
			})
		case it.Disabled:
			menu.Add(it.Label).SetEnabled(false)
		default:
			menu.Add(it.Label).OnClick(click(it.ID, t.handler(it.ID)))
		}
	}
}

// handler returns what a click on the entry with this id does.
func (t *tray) handler(id string) func() {
	switch id {
	case "set-key-top", "set-key":
		return t.onSetAPIKey
	case "show-last":
		return t.onShowLast
	case "open-config":
		return t.onOpenConfig
	case "reload":
		return t.onReload
	case "open-log":
		return t.onOpenLog
	case "about":
		return t.onAbout
	case "remove":
		return t.onRemove
	case "quit":
		return t.app.Quit
	}
	return nil
}

// SetUsage updates the running totals in the menu. The numbers are estimates.
func (t *tray) SetUsage(s usage.Summary) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.usageToday, t.usageMonth = s.Texts()
	t.rebuildLocked()
}

func click(name string, f func()) func(*application.Context) {
	return func(*application.Context) {
		slog.Info("tray menu", "entry", name)
		if f != nil {
			f()
		}
	}
}
