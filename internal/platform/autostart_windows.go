//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// autostartKey is the value name under the Run key. A variable only so that
// the test can use another name and never touch the user's real setting.
var autostartKey = "Spellcast"

// AutostartEnabled reports whether Spellcast is registered to start at logon.
func (windowsPlatform) AutostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer func() { _ = k.Close() }()
	_, _, err = k.GetStringValue(autostartKey)
	return err == nil
}

// SetAutostart adds or removes the entry in HKCU\...\Run for this executable.
func (windowsPlatform) SetAutostart(enabled bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("cannot open the Run key: %w", err)
	}
	defer func() { _ = k.Close() }()
	if !enabled {
		if err := k.DeleteValue(autostartKey); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(autostartKey, `"`+exe+`"`)
}
