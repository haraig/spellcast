package platform

import (
	"fmt"
	"strings"
)

// CFHTML wraps an HTML fragment in the "HTML Format" clipboard envelope
// (CF_HTML): a header with byte offsets, then the document. The offsets count
// UTF-8 bytes. The result has no terminating NUL; the caller adds it.
// See https://learn.microsoft.com/en-us/windows/win32/dataxchg/html-clipboard-format
func CFHTML(fragment string) []byte {
	const (
		headerFmt = "Version:0.9\r\nStartHTML:%010d\r\nEndHTML:%010d\r\nStartFragment:%010d\r\nEndFragment:%010d\r\n"
		prefix    = "<html><body>\r\n<!--StartFragment-->"
		suffix    = "<!--EndFragment-->\r\n</body></html>"
	)
	headerLen := len(fmt.Sprintf(headerFmt, 0, 0, 0, 0)) // fixed width: offsets are padded
	startHTML := headerLen
	startFragment := startHTML + len(prefix)
	endFragment := startFragment + len(fragment)
	endHTML := endFragment + len(suffix)

	var b strings.Builder
	fmt.Fprintf(&b, headerFmt, startHTML, endHTML, startFragment, endFragment)
	b.WriteString(prefix)
	b.WriteString(fragment)
	b.WriteString(suffix)
	return []byte(b.String())
}
