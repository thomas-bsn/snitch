// Applies the remembered theme and language before the first paint so the page never
// flashes the wrong theme. A plain script file because the content security policy
// forbids inline scripts. Keep the storage keys and values in sync with
// src/app/preferences.ts.
(function () {
  var root = document.documentElement;
  // Without a remembered choice, the theme follows the device.
  var theme = "system";
  var language = "en";
  try {
    var storedTheme = localStorage.getItem("snitch-ui-theme");
    if (storedTheme === "light" || storedTheme === "dark") theme = storedTheme;
    var storedLanguage = localStorage.getItem("snitch-ui-language");
    if (storedLanguage === "fr") language = storedLanguage;
  } catch (e) {
    // Storage can be blocked: the defaults apply.
  }
  var dark =
    theme === "dark" ||
    (theme === "system" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);
  root.setAttribute("data-theme", dark ? "dark" : "light");
  root.setAttribute("lang", language);
})();
