import "./style.css";
import { initAboutView } from "./about";
import { initAPIKeyView } from "./apikey";
import { initResultView } from "./result";

// One page, three views: the result window (default), the "Set API key" window
// (?view=apikey) and the About window (?view=about).
const view = new URLSearchParams(location.search).get("view");
if (view === "apikey" || view === "about") {
  document.getElementById("result-view")!.hidden = true;
  document.getElementById(`${view}-view`)!.hidden = false;
  if (view === "apikey") {
    initAPIKeyView();
  } else {
    initAboutView();
  }
} else {
  initResultView();
}
