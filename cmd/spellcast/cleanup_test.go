package main

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fakeDeps records the cleanup steps in order; fail names steps that return an error.
func fakeDeps(calls *[]string, fail ...string) cleanupDeps {
	step := func(name string) error {
		*calls = append(*calls, name)
		for _, f := range fail {
			if f == name {
				return errors.New(name + " failed")
			}
		}
		return nil
	}
	return cleanupDeps{
		DisableAutostart: func() error { return step("autostart") },
		RemoveNotifier:   func() error { return step("notifier") },
		DeleteKey:        func() error { return step("key") },
		CloseLog:         func() error { return step("close-log") },
		RemoveAll:        func(path string) error { return step("remove " + path) },
		ConfigDir:        `C:\cfg`,
		LocalDir:         `C:\local`,
	}
}

func TestCleanupWithoutPurge(t *testing.T) {
	var calls []string
	if err := cleanup(false, fakeDeps(&calls)); err != nil {
		t.Fatal(err)
	}
	if want := []string{"autostart", "notifier"}; !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestCleanupPurge(t *testing.T) {
	var calls []string
	if err := cleanup(true, fakeDeps(&calls)); err != nil {
		t.Fatal(err)
	}
	want := []string{"autostart", "notifier", "key", `remove C:\cfg`, "close-log", `remove C:\local`}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestCleanupClosesLogBeforeRemovingLocalDir(t *testing.T) {
	var calls []string
	_ = cleanup(true, fakeDeps(&calls))
	closed, removed := -1, -1
	for i, c := range calls {
		switch c {
		case "close-log":
			closed = i
		case `remove C:\local`:
			removed = i
		}
	}
	if closed < 0 || removed < 0 || closed > removed {
		t.Errorf("calls = %v: the log must be closed before its folder is removed", calls)
	}
}

func TestCleanupRunsAllStepsAndJoinsErrors(t *testing.T) {
	var calls []string
	err := cleanup(true, fakeDeps(&calls, "key", `remove C:\cfg`))
	if len(calls) != 6 {
		t.Errorf("calls = %v, want all 6 steps", calls)
	}
	if err == nil {
		t.Fatal("no error")
	}
	for _, want := range []string{"API key", `C:\cfg`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// The purge removes localDir, so the WebView2 data must live inside it.
func TestWebViewDataDirIsInsideLocalDir(t *testing.T) {
	local, err := localDir()
	if err != nil {
		t.Skip("no user cache directory:", err)
	}
	wv, err := webviewDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(local, "WebView2"); wv != want {
		t.Errorf("webviewDataDir() = %q, want %q", wv, want)
	}
}

func TestRetryRemoveAllWaitsForAShortLock(t *testing.T) {
	tries := 0
	remove := retryRemoveAll(func(string) error {
		tries++
		if tries < 3 {
			return errors.New("file in use")
		}
		return nil
	}, 5, 0)
	if err := remove(`C:\log`); err != nil || tries != 3 {
		t.Errorf("err = %v after %d tries, want success on the 3rd", err, tries)
	}
}

func TestRetryRemoveAllGivesUp(t *testing.T) {
	tries := 0
	remove := retryRemoveAll(func(string) error { tries++; return errors.New("file in use") }, 4, 0)
	if err := remove(`C:\log`); err == nil || tries != 4 {
		t.Errorf("err = %v after %d tries, want an error after 4", err, tries)
	}
}
