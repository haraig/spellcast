// Run with: npm test (Node's built-in test runner, no extra packages).
import { test } from "node:test";
import assert from "node:assert/strict";
import { diffWords, hasChanges, tokenize } from "./diff.ts";

/** Compact view: "=word +word -word". */
function show(before: string, after: string): string {
  const sign = { equal: "=", insert: "+", delete: "-" };
  return diffWords(before, after)
    .map((p) => sign[p.op] + p.text)
    .join(" ");
}

test("tokenize keeps words, apostrophes, hyphens and punctuation apart", () => {
  assert.deepEqual(
    tokenize("Don't stop, well-known!").map((t) => t.text),
    ["Don't", "stop", ",", "well-known", "!"],
  );
});

test("tokenize handles umlauts, digits and emoji", () => {
  assert.deepEqual(
    tokenize("Grüße 42 👋").map((t) => t.text),
    ["Grüße", "42", "👋"],
  );
});

test("identical texts have no changes", () => {
  assert.equal(hasChanges(diffWords("Hello world.", "Hello world.")), false);
});

test("whitespace differences are not changes", () => {
  assert.equal(hasChanges(diffWords("a  b\r\nc", "a b\nc")), false);
});

test("a replaced word shows the deletion first", () => {
  assert.equal(show("I has a dog.", "I have a dog."), "=I -has +have =a =dog =.");
});

test("an inserted word", () => {
  assert.equal(show("very good", "very, very good"), "=very +, +very =good");
});

test("a deleted word", () => {
  assert.equal(show("a big red car", "a red car"), "=a -big =red =car");
});

test("empty before: everything is inserted, empty after: everything is deleted", () => {
  assert.equal(show("", "new text"), "+new +text");
  assert.equal(show("old text", ""), "-old -text");
});

test("equal pieces keep the whitespace of the answer", () => {
  const p = diffWords("a b", "a\n\nb");
  assert.equal(p[0].space, "\n\n");
});

test("a long text still produces a result (fallback)", () => {
  const before = Array.from({ length: 2500 }, (_, i) => "a" + i).join(" ");
  const after = Array.from({ length: 2500 }, (_, i) => "b" + i).join(" ");
  const p = diffWords(before, after);
  assert.equal(p.filter((x) => x.op === "delete").length, 2500);
  assert.equal(p.filter((x) => x.op === "insert").length, 2500);
});
