import { Events, Window } from "@wailsio/runtime";

// Match KeyStatePayload and KeyResultPayload in internal/ui/apikey.go.
interface StatePayload {
  source: string;
}
interface ResultPayload {
  ok: boolean;
  message: string;
}

function payload<T>(event: { data: unknown }): T {
  return (Array.isArray(event.data) ? event.data[0] : event.data) as T;
}

const $ = <T extends HTMLElement>(id: string) => document.getElementById(id) as T;

export function initAPIKeyView() {
  const form = $<HTMLFormElement>("apikey-form");
  const input = $<HTMLInputElement>("apikey-input");
  const current = $("apikey-current");
  const message = $("apikey-message");
  const saveBtn = $<HTMLButtonElement>("apikey-save");
  const closeBtn = $<HTMLButtonElement>("apikey-cancel");

  function setMessage(text: string, kind: "" | "ok" | "error") {
    message.textContent = text;
    message.className = kind;
  }

  Events.On("apikey:state", (e) => {
    const { source } = payload<StatePayload>(e);
    current.textContent = source ? `Current key: ${source}.` : "No key set yet.";
    setMessage("", "");
    input.value = "";
    saveBtn.disabled = false;
    input.focus();
  });

  Events.On("apikey:result", (e) => {
    const { ok, message: text } = payload<ResultPayload>(e);
    setMessage(text, ok ? "ok" : "error");
    saveBtn.disabled = false;
    if (ok) {
      input.value = ""; // do not keep the key around
      current.textContent = "Current key: credential store.";
      window.setTimeout(() => void Window.Close(), 1200);
    }
  });

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const key = input.value.trim();
    if (key === "") {
      setMessage("Enter a key first.", "error");
      return;
    }
    saveBtn.disabled = true;
    setMessage("Checking the key…", "");
    void Events.Emit("apikey:save", { key });
  });

  closeBtn.addEventListener("click", () => void Window.Close());
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      void Window.Close();
    }
  });
}
