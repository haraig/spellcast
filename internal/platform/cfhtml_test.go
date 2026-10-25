package platform

import (
	"strconv"
	"strings"
	"testing"
)

func TestCFHTMLOffsets(t *testing.T) {
	frag := "<p>Grüße 👋 <b>fett</b></p>"
	data := string(CFHTML(frag))

	offset := func(name string) int {
		i := strings.Index(data, name+":")
		if i < 0 {
			t.Fatalf("header %s missing", name)
		}
		n, err := strconv.Atoi(data[i+len(name)+1 : i+len(name)+11])
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	if got := data[offset("StartFragment"):offset("EndFragment")]; got != frag {
		t.Errorf("fragment by offsets = %q, want %q", got, frag)
	}
	if offset("EndHTML") != len(data) {
		t.Errorf("EndHTML = %d, want %d", offset("EndHTML"), len(data))
	}
	if !strings.HasPrefix(data[offset("StartHTML"):], "<html>") {
		t.Error("StartHTML does not point at <html>")
	}
}
