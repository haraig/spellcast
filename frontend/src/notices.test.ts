// Run with: npm test (Node's built-in test runner, no extra packages).
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { parseNotices } from "./notices.ts";

const SEP = "=".repeat(80);

test("parseNotices splits the intro and the entries", () => {
  const text = [
    "Spellcast includes the following third-party software.",
    "",
    SEP,
    "github.com/yuin/goldmark (MIT)",
    SEP,
    "",
    "MIT License",
    "",
    SEP,
    "dompurify (MPL-2.0 OR Apache-2.0)",
    SEP,
    "",
    "Apache text",
    "",
    "MPL text",
    "",
  ].join("\n");
  assert.deepEqual(parseNotices(text), {
    intro: "Spellcast includes the following third-party software.",
    notices: [
      { name: "github.com/yuin/goldmark", license: "MIT", text: "MIT License" },
      { name: "dompurify", license: "MPL-2.0 OR Apache-2.0", text: "Apache text\n\nMPL text" },
    ],
  });
});

test("parseNotices keeps separator lines inside a license text", () => {
  const text = [SEP, "a (MIT)", SEP, "", "line", SEP, "not a header", "more", ""].join("\n");
  assert.deepEqual(parseNotices(text).notices, [
    { name: "a", license: "MIT", text: ["line", SEP, "not a header", "more"].join("\n") },
  ]);
});

test("parseNotices handles CRLF line endings", () => {
  const text = [SEP, "a (MIT)", SEP, "", "line", ""].join("\r\n");
  assert.deepEqual(parseNotices(text).notices, [{ name: "a", license: "MIT", text: "line" }]);
});

test("parseNotices reads every entry of THIRD_PARTY_NOTICES.txt", () => {
  const text = readFileSync(new URL("../../THIRD_PARTY_NOTICES.txt", import.meta.url), "utf8");
  const { intro, notices } = parseNotices(text);
  assert.match(intro, /^Spellcast includes/);
  const headers = text.split("\n").filter((l, i, a) => a[i - 1]?.startsWith("====") && a[i + 1]?.startsWith("===="));
  assert.equal(notices.length, headers.length);
  for (const n of notices) {
    assert.ok(n.license, `${n.name}: no license`);
    assert.ok(n.text, `${n.name}: no text`);
  }
  assert.ok(notices.some((n) => n.name === "marked" && n.license === "MIT"));
});
