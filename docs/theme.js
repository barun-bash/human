// theme.js — Dark/light theme toggle for human_ docs
// FOUC prevention: each page must include this inline in <head>:
//   <script>(function(){var t=localStorage.getItem('human-theme');if(t)document.documentElement.setAttribute('data-theme',t);else if(window.matchMedia&&window.matchMedia('(prefers-color-scheme:light)').matches)document.documentElement.setAttribute('data-theme','light')})()</script>

(function () {
  var toggle = document.getElementById('themeToggle');
  if (!toggle) return;

  // Dark favicon (dark bg, light "h", orange "_")
  var darkFavicon = "data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'><rect width='32' height='32' rx='6' fill='%230D0D0D'/><text x='4' y='24' font-family='sans-serif' font-weight='700' font-size='22' fill='%23F5F5F3'>h</text><text x='18' y='24' font-family='sans-serif' font-weight='700' font-size='22' fill='%23E85D3A'>_</text></svg>";
  // Light favicon (light bg with border, dark "h", orange "_")
  var lightFavicon = "data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'><rect width='32' height='32' rx='6' fill='%23FAFAF8' stroke='%23E0E0DC' stroke-width='1'/><text x='4' y='24' font-family='sans-serif' font-weight='700' font-size='22' fill='%231A1A1A'>h</text><text x='18' y='24' font-family='sans-serif' font-weight='700' font-size='22' fill='%23D04E2D'>_</text></svg>";

  function setFavicon(theme) {
    var link = document.querySelector("link[rel='icon']");
    if (link) link.href = theme === 'light' ? lightFavicon : darkFavicon;
  }

  // Set favicon on load based on current theme
  var current = document.documentElement.getAttribute('data-theme');
  setFavicon(current);

  toggle.addEventListener('click', function () {
    var html = document.documentElement;
    var now = html.getAttribute('data-theme');
    var next = now === 'light' ? 'dark' : 'light';
    html.setAttribute('data-theme', next);
    localStorage.setItem('human-theme', next);
    setFavicon(next);
  });
})();
