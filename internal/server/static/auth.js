/* PROPhish auth pages (login / setup) — same-origin progressive enhancement.
   The strict CSP blocks inline scripts, so the theme toggle and the password
   reveal live here. The theme choice shares the 'prophish-theme' localStorage
   key with the main app, so a preference set here carries into the console. The
   active-theme ICON is chosen in CSS (see login.html), so this file never paints. */
(function () {
  var root = document.documentElement;

  // Apply a saved theme as early as possible (this script is in <head>).
  try {
    var saved = localStorage.getItem('prophish-theme');
    if (saved === 'dark' || saved === 'light') root.setAttribute('data-theme', saved);
  } catch (e) {}

  function systemDark() {
    return !!(window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches);
  }
  function effective() {
    var a = root.getAttribute('data-theme');
    if (a === 'dark' || a === 'light') return a;
    return systemDark() ? 'dark' : 'light';
  }
  function wireTheme() {
    var btn = document.getElementById('theme-toggle');
    if (!btn) return;
    btn.addEventListener('click', function () {
      var next = effective() === 'dark' ? 'light' : 'dark';
      root.setAttribute('data-theme', next);
      try { localStorage.setItem('prophish-theme', next); } catch (e) {}
    });
  }
  // Password reveal: a <button data-pw-toggle="<input id>"> flips that field
  // between password and text, and toggles its own .on class for the icon state.
  function wirePasswordToggles() {
    var btns = document.querySelectorAll('[data-pw-toggle]');
    for (var i = 0; i < btns.length; i++) {
      (function (btn) {
        btn.addEventListener('click', function () {
          var inp = document.getElementById(btn.getAttribute('data-pw-toggle'));
          if (!inp) return;
          var show = inp.type === 'password';
          inp.type = show ? 'text' : 'password';
          btn.classList.toggle('on', show);
          btn.setAttribute('aria-pressed', show ? 'true' : 'false');
        });
      })(btns[i]);
    }
  }

  function init() { wireTheme(); wirePasswordToggles(); }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
  else init();
})();
