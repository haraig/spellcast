import { Browser, Clipboard, Events, Window } from "@wailsio/runtime";
import { diffWords, hasChanges, type Piece } from "./diff";
import { renderMarkdown } from "./markdown";

// Match core.View and the payloads in internal/ui/window.go.
interface BeginPayload {
  id: number;
  action: string;
  source: string;
  output: "plain" | "markdown";
  original: string; // the selected text, for "Show changes"
  canReplace: boolean;
  replaceReason: string;
}
interface ReplaceStatePayload {
  id: number;
  canReplace: boolean;
  reason: string;
}
interface DeltaPayload {
  id: number;
  text: string;
}
interface DonePayload {
  id: number;
  stopReason: string;
  usage: string;
}
interface FailPayload {
  id: number;
  message: string;
}

const NOTE_TRUNCATED = "Answer truncated (token limit reached).";
const NOTE_REFUSAL = "The model declined to answer this request.";
const NOTE_EMPTY = "The answer is empty.";
const NOTE_NO_CHANGES = "No changes: the answer has the same words as the selection.";
const NOTE_CANCELLED = "The request was cancelled. The answer is incomplete.";

/** One value emitted from Go arrives as data; several would arrive as an array. */
function payload<T>(event: { data: unknown }): T {
  return (Array.isArray(event.data) ? event.data[0] : event.data) as T;
}

const $ = <T extends HTMLElement>(id: string) => document.getElementById(id) as T;

/** Equal words as text, deleted and inserted runs as <del> and <ins>. */
function changeNodes(pieces: Piece[]): Node[] {
  const nodes: Node[] = [];
  for (let i = 0; i < pieces.length; ) {
    const op = pieces[i].op;
    let text = "";
    let last = "";
    for (; i < pieces.length && pieces[i].op === op; i++) {
      text += last + pieces[i].text;
      last = op === "delete" ? " " : pieces[i].space; // the old line breaks do not matter
    }
    if (op === "equal") {
      nodes.push(document.createTextNode(text));
    } else {
      const el = document.createElement(op === "delete" ? "del" : "ins");
      el.textContent = text;
      nodes.push(el);
    }
    nodes.push(document.createTextNode(last));
  }
  return nodes;
}

export function initResultView() {
  const actionEl = $("action");
  const sourceEl = $("source");
  const answerEl = $("answer");
  const notesEl = $("notes");
  const scrollEl = $("answer-scroll");
  const spinnerEl = $("spinner");
  const noteEl = $("note");
  const usageEl = $("usage");
  const errorEl = $("error");
  const copyBtn = $<HTMLButtonElement>("copy");
  const changesBtn = $<HTMLButtonElement>("changes");
  const noChangesEl = $("no-changes");
  const replaceBtn = $<HTMLButtonElement>("replace");
  const closeBtn = $<HTMLButtonElement>("close");

  // The frontend always keeps the raw model output. Copy (and later Replace)
  // use it, never the rendered HTML.
  let sessionId = 0;
  let raw = ""; // the usable text; for two-part answers the notes are kept apart
  let notesRaw = "";
  let original = ""; // the selected text
  let showChanges = false; // the user's choice; kept from one session to the next
  let output: "plain" | "markdown" = "plain";
  let done = false; // the answer is complete
  let stopReason = "";
  let canReplace = false; // the target allows Replace (set by Go)
  let replaceReason = "";
  let renderQueued = false;
  let copyTimer: number | undefined;

  function render() {
    renderQueued = false;
    const nearBottom = scrollEl.scrollHeight - scrollEl.scrollTop - scrollEl.clientHeight < 40;
    noChangesEl.hidden = true;
    if (changesAvailable() && showChanges) {
      renderChanges();
    } else if (output === "markdown") {
      answerEl.className = "markdown";
      answerEl.innerHTML = renderMarkdown(raw);
    } else {
      answerEl.className = "plain";
      answerEl.textContent = raw; // no HTML interpretation
    }
    renderNotes();
    if (nearBottom) {
      scrollEl.scrollTop = scrollEl.scrollHeight; // follow the stream unless the user scrolled up
    }
  }

  /** "Show changes" needs a complete answer and a selection to compare with. */
  function changesAvailable(): boolean {
    return done && stopReason !== "cancelled" && original.trim() !== "" && raw.trim() !== "";
  }

  function updateChangesButton() {
    const available = changesAvailable();
    changesBtn.disabled = !available;
    changesBtn.setAttribute("aria-pressed", String(available && showChanges));
    changesBtn.title = available
      ? "Highlight what changed against the selected text (Ctrl+D)"
      : "Available when the answer is complete.";
  }

  function toggleChanges() {
    if (!changesAvailable()) return;
    showChanges = !showChanges;
    updateChangesButton();
    render();
    scrollEl.scrollTop = 0;
  }

  /**
   * Word-level diff of the selection and the answer, local, no API call. Shown
   * as plain text even for Markdown actions, because the diff compares the
   * raw Markdown. Built from DOM nodes with textContent, never as HTML.
   */
  function renderChanges() {
    const pieces = diffWords(original, raw);
    answerEl.className = "plain changes";
    answerEl.replaceChildren(...changeNodes(pieces));
    noChangesEl.textContent = NOTE_NO_CHANGES;
    noChangesEl.hidden = hasChanges(pieces);
  }

  /** Notes of a two-part answer: shown below the text, muted, same output type. */
  function renderNotes() {
    const text = notesRaw.trim();
    notesEl.hidden = text === "";
    if (text === "") return;
    if (output === "markdown") {
      notesEl.innerHTML = renderMarkdown(text);
    } else {
      notesEl.textContent = text;
    }
  }

  function scheduleRender() {
    if (!renderQueued) {
      renderQueued = true;
      requestAnimationFrame(render);
    }
  }

  /** Replace needs a complete, non-empty answer and a target that allows it. */
  function replaceEnabled(): boolean {
    return canReplace && done && stopReason !== "cancelled" && raw.trim() !== "";
  }

  function updateReplaceButton() {
    const enabled = replaceEnabled();
    replaceBtn.disabled = !enabled;
    if (!canReplace) {
      replaceBtn.title = replaceReason || "Replace is not possible here.";
    } else if (!done) {
      replaceBtn.title = "Available when the answer is complete.";
    } else if (enabled) {
      replaceBtn.title = "Replace the selected text with this answer (Ctrl+Enter)";
    } else {
      replaceBtn.title = "There is no complete answer to paste.";
    }
  }

  function requestReplace() {
    if (!replaceEnabled()) return;
    void Events.Emit("session:replace", { id: sessionId });
  }

  function showNote(text: string) {
    noteEl.textContent = text;
    noteEl.hidden = false;
  }

  function begin(p: BeginPayload) {
    sessionId = p.id;
    raw = "";
    original = p.original ?? "";
    notesRaw = "";
    notesEl.hidden = true;
    notesEl.textContent = "";
    notesEl.className = p.output === "markdown" ? "notes markdown" : "notes plain";
    done = false;
    stopReason = "";
    canReplace = p.canReplace;
    replaceReason = p.replaceReason ?? "";
    output = p.output === "markdown" ? "markdown" : "plain";
    actionEl.textContent = p.action;
    sourceEl.textContent = p.source ? `from ${p.source}` : "";
    answerEl.className = output === "markdown" ? "markdown" : "plain";
    answerEl.textContent = "";
    noteEl.hidden = true;
    usageEl.hidden = true;
    usageEl.textContent = "";
    errorEl.hidden = true;
    spinnerEl.hidden = false;
    copyBtn.disabled = true;
    updateReplaceButton();
    updateChangesButton();
    scrollEl.scrollTop = 0;
  }

  function resetCopyLabel() {
    copyBtn.textContent = "Copy";
  }

  async function copyRaw() {
    if (raw === "") return;
    try {
      await Clipboard.SetText(raw);
      copyBtn.textContent = "Copied";
      window.clearTimeout(copyTimer);
      copyTimer = window.setTimeout(resetCopyLabel, 1200);
    } catch (e) {
      console.error("copy failed", e);
      copyBtn.textContent = "Copy failed";
      window.clearTimeout(copyTimer);
      copyTimer = window.setTimeout(resetCopyLabel, 2000);
    }
  }

  Events.On("session:begin", (e) => begin(payload<BeginPayload>(e)));

  Events.On("session:delta", (e) => {
    const p = payload<DeltaPayload>(e);
    if (p.id !== sessionId) return; // a replaced session
    spinnerEl.hidden = true;
    raw += p.text;
    copyBtn.disabled = false;
    scheduleRender();
  });

  Events.On("session:notes", (e) => {
    const p = payload<DeltaPayload>(e);
    if (p.id !== sessionId) return;
    spinnerEl.hidden = true;
    notesRaw += p.text;
    scheduleRender();
  });

  Events.On("session:done", (e) => {
    const p = payload<DonePayload>(e);
    if (p.id !== sessionId) return;
    spinnerEl.hidden = true;
    done = true;
    stopReason = p.stopReason;
    render();
    updateReplaceButton();
    updateChangesButton();
    if (p.usage) {
      usageEl.textContent = p.usage;
      usageEl.hidden = false;
    }
    if (p.stopReason === "cancelled") {
      showNote(NOTE_CANCELLED);
    } else if (p.stopReason === "max_tokens") {
      showNote(NOTE_TRUNCATED);
    } else if (p.stopReason === "refusal") {
      showNote(NOTE_REFUSAL);
    } else if (raw.trim() === "") {
      showNote(NOTE_EMPTY);
    }
  });

  Events.On("session:fail", (e) => {
    const p = payload<FailPayload>(e);
    if (p.id !== sessionId) return;
    spinnerEl.hidden = true;
    render();
    errorEl.textContent = p.message;
    errorEl.hidden = false;
  });

  Events.On("session:replace-state", (e) => {
    const p = payload<ReplaceStatePayload>(e);
    if (p.id !== sessionId) return;
    canReplace = p.canReplace;
    replaceReason = p.reason;
    updateReplaceButton();
  });

  copyBtn.addEventListener("click", () => void copyRaw());
  changesBtn.addEventListener("click", toggleChanges);
  replaceBtn.addEventListener("click", requestReplace);
  closeBtn.addEventListener("click", () => void Window.Close());

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      void Window.Close();
    } else if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      requestReplace();
    } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "d") {
      e.preventDefault();
      toggleChanges();
    } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "c") {
      // Selected text in the window: the normal copy. Otherwise the whole raw answer.
      const selection = window.getSelection()?.toString() ?? "";
      if (selection === "" && raw !== "") {
        e.preventDefault();
        void copyRaw();
      }
    }
  });

  // Links go to the default browser, never into this window.
  answerEl.addEventListener("click", (e) => {
    const link = (e.target as HTMLElement).closest("a");
    if (!link) return;
    e.preventDefault();
    const href = link.getAttribute("href") ?? "";
    if (/^https?:\/\//i.test(href)) {
      void Browser.OpenURL(href);
    }
  });
}
