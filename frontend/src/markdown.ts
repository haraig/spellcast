import DOMPurify from "dompurify";
import { marked } from "marked";

marked.setOptions({ gfm: true, breaks: false });

// Open links in rel="noopener" mode; the click handler sends them to the browser.
DOMPurify.addHook("afterSanitizeAttributes", (node) => {
  if (node.tagName === "A") {
    node.setAttribute("rel", "noopener noreferrer");
  }
});

/** Renders Markdown to sanitized HTML. Unfinished input (while streaming) is fine. */
export function renderMarkdown(raw: string): string {
  const html = marked.parse(raw, { async: false }) as string;
  return DOMPurify.sanitize(html);
}
