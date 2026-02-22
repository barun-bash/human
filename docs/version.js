// Single source of truth for version info across all docs pages.
// Update this file when releasing a new version — all pages pick it up automatically.
var HUMAN_VERSION = "0.4.2";
var HUMAN_VERSION_TAG = "v" + HUMAN_VERSION;
var HUMAN_RELEASE_URL = "https://github.com/barun-bash/human/releases/tag/" + HUMAN_VERSION_TAG;
var HUMAN_INSTALL_CMD = "go install github.com/barun-bash/human/cmd/human@" + HUMAN_VERSION_TAG;
var HUMAN_STATUS = "Language spec, LLM prompt, 8 examples, 14 generators. 85+ files per build.";

document.addEventListener("DOMContentLoaded", function () {
  // Populate elements with data-version attributes
  document.querySelectorAll("[data-version]").forEach(function (el) {
    var kind = el.getAttribute("data-version");
    if (kind === "tag") el.textContent = HUMAN_VERSION_TAG + " released";
    if (kind === "link") el.href = HUMAN_RELEASE_URL;
    if (kind === "status") el.textContent = HUMAN_STATUS;
    if (kind === "install") el.textContent = el.textContent.replace(/@v[\d.]+/, "@" + HUMAN_VERSION_TAG);
  });
});
