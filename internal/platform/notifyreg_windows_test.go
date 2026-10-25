//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestRemoveNotificationRegistration(t *testing.T) {
	const app = "spellcast-test-notifyreg"
	const guid = "{00000000-1111-2222-3333-444444444444}"
	mustCreate := func(path string) registry.Key {
		k, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
		if err != nil {
			t.Fatalf("create %s: %v", path, err)
		}
		return k
	}
	k := mustCreate(`Software\Classes\AppUserModelId\` + app)
	if err := k.SetStringValue("CustomActivator", guid); err != nil {
		t.Fatal(err)
	}
	_ = k.Close()
	_ = mustCreate(`Software\Classes\CLSID\` + guid + `\LocalServer32`).Close()
	_ = mustCreate(`SOFTWARE\` + app + `\NotificationCategories`).Close()
	icon := filepath.Join(os.TempDir(), app+guid+".png")
	if err := os.WriteFile(icon, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RemoveNotificationRegistration(app); err != nil {
		t.Fatalf("remove: %v", err)
	}
	for _, path := range []string{`Software\Classes\AppUserModelId\` + app, `Software\Classes\CLSID\` + guid, `SOFTWARE\` + app} {
		if k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE); err == nil {
			_ = k.Close()
			t.Errorf("%s still exists", path)
		}
	}
	if _, err := os.Stat(icon); !os.IsNotExist(err) {
		t.Errorf("icon %s still exists (err %v)", icon, err)
	}
	// Nothing registered: not an error.
	if err := RemoveNotificationRegistration(app); err != nil {
		t.Errorf("second remove: %v", err)
	}
}
