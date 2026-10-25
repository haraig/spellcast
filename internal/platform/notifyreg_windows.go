//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// RemoveNotificationRegistration deletes what the Wails notification service
// registers for app under HKCU (the AppUserModelId entry, the COM activator
// that points to the exe, the notification categories) and its icon copy in
// %TEMP%. Missing entries are not an error. Call it only when the
// notification service is not running, or it writes the entries again.
func RemoveNotificationRegistration(app string) error {
	var errs []error
	aumid := `Software\Classes\AppUserModelId\` + app
	if k, err := registry.OpenKey(registry.CURRENT_USER, aumid, registry.QUERY_VALUE); err == nil {
		guid, _, gerr := k.GetStringValue("CustomActivator")
		_ = k.Close()
		if gerr == nil && guid != "" {
			errs = append(errs, deleteKeyTree(`Software\Classes\CLSID\`+guid))
			if err := os.Remove(filepath.Join(os.TempDir(), app+guid+".png")); err != nil && !errors.Is(err, os.ErrNotExist) {
				errs = append(errs, err)
			}
		}
	}
	errs = append(errs, deleteKeyTree(aumid), deleteKeyTree(`SOFTWARE\`+app))
	return errors.Join(errs...)
}

// deleteKeyTree deletes an HKCU key with all its subkeys. A missing key is fine.
func deleteKeyTree(path string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.ENUMERATE_SUB_KEYS)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	subs, err := k.ReadSubKeyNames(-1)
	_ = k.Close()
	if err != nil {
		return fmt.Errorf("list %s: %w", path, err)
	}
	for _, s := range subs {
		if err := deleteKeyTree(path + `\` + s); err != nil {
			return err
		}
	}
	if err := registry.DeleteKey(registry.CURRENT_USER, path); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("delete %s: %w", path, err)
	}
	return nil
}
