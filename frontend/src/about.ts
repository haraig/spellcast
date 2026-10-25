import { Events, Window } from "@wailsio/runtime";
import iconUrl from "../../build/appicon.svg";

// Matches AboutInfo in internal/ui/about.go.
interface AboutInfo {
  version: string;
  copyright: string;
  license: string;
  repoUrl: string;
  notices: string;
}

function payload<T>(event: { data: unknown }): T {
  return (Array.isArray(event.data) ? event.data[0] : event.data) as T;
}

const $ = <T extends HTMLElement>(id: string) => document.getElementById(id) as T;

export function initAboutView() {
  const main = $("about-main");
  const licenses = $("about-licenses");
  const notices = $("about-notices");

  $<HTMLImageElement>("about-icon").src = iconUrl;

  function showLicenses(on: boolean) {
    main.hidden = on;
    licenses.hidden = !on;
    (on ? $("about-back") : $("about-show-licenses")).focus();
  }

  Events.On("about:info", (e) => {
    const info = payload<AboutInfo>(e);
    $("about-version").textContent = info.version;
    $("about-copyright").textContent = info.copyright;
    $("about-license").textContent = info.license;
    $("about-repo").textContent = info.repoUrl.replace(/^https:\/\//, "");
    notices.textContent = info.notices;
    notices.scrollTop = 0;
    showLicenses(false);
  });

  $("about-repo").addEventListener("click", (e) => {
    e.preventDefault();
    void Events.Emit("about:open-repo");
  });
  $("about-show-licenses").addEventListener("click", () => showLicenses(true));
  $("about-back").addEventListener("click", () => showLicenses(false));
  for (const id of ["about-close", "about-close-2"]) {
    $(id).addEventListener("click", () => void Window.Close());
  }
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      void Window.Close();
    }
  });
}
