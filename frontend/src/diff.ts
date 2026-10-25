// Word-level diff between the selected text and the answer ("Show changes").
// Pure functions, no DOM; tests are in diff.test.ts.

export type Op = "equal" | "insert" | "delete";

export interface Piece {
  op: Op;
  text: string; // a word or a punctuation mark
  space: string; // the whitespace that follows it in its own text
}

// Beyond this many table cells (after trimming) the exact diff is skipped and
// the middle part is shown as one deletion and one insertion.
const MAX_CELLS = 4_000_000;

interface Token {
  text: string;
  space: string;
}

/** Words (letters, digits, apostrophes, hyphens inside a word) and single punctuation marks, each with the whitespace after it. */
export function tokenize(s: string): Token[] {
  const tokens: Token[] = [];
  const re = /([\p{L}\p{N}_]+(?:['’\-][\p{L}\p{N}_]+)*|[^\s\p{L}\p{N}_])(\s*)/gu;
  // Leading whitespace belongs to nobody; it is not part of the comparison.
  for (const m of s.matchAll(re)) {
    tokens.push({ text: m[1], space: m[2] });
  }
  return tokens;
}

/**
 * Compares `before` (the selection) with `after` (the answer) word by word.
 * Whitespace is not compared, so a changed line break or a doubled space is no
 * change. Equal and inserted pieces carry the whitespace of the answer, deleted
 * pieces the whitespace of the selection.
 */
export function diffWords(before: string, after: string): Piece[] {
  const a = tokenize(before);
  const b = tokenize(after);

  let start = 0;
  while (start < a.length && start < b.length && a[start].text === b[start].text) start++;
  let endA = a.length;
  let endB = b.length;
  while (endA > start && endB > start && a[endA - 1].text === b[endB - 1].text) {
    endA--;
    endB--;
  }

  const out: Piece[] = [];
  for (let i = 0; i < start; i++) out.push({ op: "equal", text: b[i].text, space: b[i].space });
  out.push(...diffMiddle(a.slice(start, endA), b.slice(start, endB)));
  for (let i = endB; i < b.length; i++) out.push({ op: "equal", text: b[i].text, space: b[i].space });
  return out;
}

function diffMiddle(a: Token[], b: Token[]): Piece[] {
  const n = a.length;
  const m = b.length;
  const del = (t: Token): Piece => ({ op: "delete", text: t.text, space: t.space });
  const ins = (t: Token): Piece => ({ op: "insert", text: t.text, space: t.space });
  if (n === 0) return b.map(ins);
  if (m === 0) return a.map(del);
  if ((n + 1) * (m + 1) > MAX_CELLS) return [...a.map(del), ...b.map(ins)];

  // lcs[i][j]: length of the longest common subsequence of a[i..] and b[j..].
  const w = m + 1;
  const lcs = new Uint32Array((n + 1) * w);
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lcs[i * w + j] =
        a[i].text === b[j].text
          ? lcs[(i + 1) * w + j + 1] + 1
          : Math.max(lcs[(i + 1) * w + j], lcs[i * w + j + 1]);
    }
  }

  const out: Piece[] = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (a[i].text === b[j].text) {
      out.push({ op: "equal", text: b[j].text, space: b[j].space });
      i++;
      j++;
    } else if (lcs[(i + 1) * w + j] >= lcs[i * w + j + 1]) {
      out.push(del(a[i++])); // deletions first, so a replaced word reads "old new"
    } else {
      out.push(ins(b[j++]));
    }
  }
  while (i < n) out.push(del(a[i++]));
  while (j < m) out.push(ins(b[j++]));
  return out;
}

/** True if the two texts differ in at least one word or punctuation mark. */
export function hasChanges(pieces: Piece[]): boolean {
  return pieces.some((p) => p.op !== "equal");
}
