// Splits THIRD_PARTY_NOTICES.txt (written by scripts/gen-notices.sh) into
// its entries for the About window. Pure functions, no DOM; tests are in
// notices.test.ts.

export interface Notice {
  name: string; // "github.com/yuin/goldmark"
  license: string; // SPDX expression, "MIT" or "MPL-2.0 OR Apache-2.0"
  text: string; // the license text
}

// An entry starts with a separator line, a "NAME (LICENSE)" line and another
// separator line.
const SEP = /^={20,}$/;
const HEADER = /^(.+) \((.+)\)$/;

export function parseNotices(text: string): { intro: string; notices: Notice[] } {
  const lines = text.replace(/\r\n/g, "\n").split("\n");
  const intro: string[] = [];
  const entries: { name: string; license: string; lines: string[] }[] = [];
  let body = intro;
  for (let i = 0; i < lines.length; i++) {
    const m = SEP.test(lines[i]) && SEP.test(lines[i + 2] ?? "") ? HEADER.exec(lines[i + 1]) : null;
    if (m) {
      body = [];
      entries.push({ name: m[1], license: m[2], lines: body });
      i += 2;
    } else {
      body.push(lines[i]);
    }
  }
  return {
    intro: intro.join("\n").trim(),
    notices: entries.map((e) => ({ name: e.name, license: e.license, text: e.lines.join("\n").trim() })),
  };
}
