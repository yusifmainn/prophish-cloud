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
  function reduceMotion() {
    return !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches);
  }
  function flipTheme(next) {
    root.setAttribute('data-theme', next);
    try { localStorage.setItem('prophish-theme', next); } catch (e) {}
  }
  function wireTheme() {
    var btn = document.getElementById('theme-toggle');
    if (!btn) return;
    btn.addEventListener('click', function () {
      var next = effective() === 'dark' ? 'light' : 'dark';
      // Circular reveal growing from the toggle (View Transitions API); fall back
      // to a colour cross-fade, then an instant flip.
      if (reduceMotion() || typeof document.startViewTransition !== 'function') {
        if (!reduceMotion()) {
          root.classList.add('theme-anim');
          window.setTimeout(function () { root.classList.remove('theme-anim'); }, 600);
        }
        flipTheme(next);
        return;
      }
      var r = btn.getBoundingClientRect();
      var x = r.left + r.width / 2, y = r.top + r.height / 2;
      var end = Math.hypot(Math.max(x, window.innerWidth - x), Math.max(y, window.innerHeight - y));
      var vt = document.startViewTransition(function () { flipTheme(next); });
      vt.ready.then(function () {
        root.animate(
          { clipPath: ['circle(0px at ' + x + 'px ' + y + 'px)', 'circle(' + end + 'px at ' + x + 'px ' + y + 'px)'] },
          { duration: 540, easing: 'ease-in-out', pseudoElement: '::view-transition-new(root)' }
        );
      }).catch(function () {});
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
