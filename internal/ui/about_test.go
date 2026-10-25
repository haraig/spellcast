package ui

import (
	"encoding/json"
	"testing"
)

func TestAboutInfoJSON(t *testing.T) {
	data, err := json.Marshal(AboutInfo{Version: "v", Copyright: "c", License: "l", RepoURL: "r", Notices: "n"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"version": "v", "copyright": "c", "license": "l", "repoUrl": "r", "notices": "n"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q (JSON: %s)", k, got[k], v, data)
		}
	}
}
