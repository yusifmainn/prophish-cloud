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
  function flipTheme(next) {
    root.setAttribute('data-theme', next);
    try { localStorage.setItem('prophish-theme', next); } catch (e) {}
  }
  // Circular reveal in ANY browser (no View Transitions needed): a circle of the
  // NEW page colour grows from the control, then the theme flips as it completes.
  function overlayReveal(next, x, y) {
    var prev = root.getAttribute('data-theme');
    root.setAttribute('data-theme', next);
    var newBg = window.getComputedStyle(document.body).backgroundColor || '#ffffff';
    if (prev === 'dark' || prev === 'light') root.setAttribute('data-theme', prev);
    else root.removeAttribute('data-theme');
    var end = Math.hypot(Math.max(x, window.innerWidth - x), Math.max(y, window.innerHeight - y));
    var ov = document.createElement('div');
    ov.setAttribute('aria-hidden', 'true');
    ov.style.cssText = 'position:fixed;inset:0;z-index:2147483647;pointer-events:none;background:' + newBg;
    ov.style.clipPath = 'circle(0px at ' + x + 'px ' + y + 'px)';
    document.body.appendChild(ov);
    var done = false;
    function finish() { if (done) return; done = true; flipTheme(next); if (ov.parentNode) ov.parentNode.removeChild(ov); }
    try {
      ov.animate(
        { clipPath: ['circle(0px at ' + x + 'px ' + y + 'px)', 'circle(' + end + 'px at ' + x + 'px ' + y + 'px)'] },
        { duration: 520, easing: 'ease-in-out' }
      ).onfinish = finish;
    } catch (e) { finish(); return; }
    window.setTimeout(finish, 680);
  }
  function wireTheme() {
    var btn = document.getElementById('theme-toggle');
    if (!btn) return;
    btn.addEventListener('click', function () {
      var next = effective() === 'dark' ? 'light' : 'dark';
      var r = btn.getBoundingClientRect();
      var x = r.left + r.width / 2, y = r.top + r.height / 2;
      // View Transitions (reveals the real new content) where available, else the
      // colour-circle overlay. Always animates, growing from the toggle.
      if (typeof document.startViewTransition === 'function') {
        var end = Math.hypot(Math.max(x, window.innerWidth - x), Math.max(y, window.innerHeight - y));
        var vt = document.startViewTransition(function () { flipTheme(next); });
        vt.ready.then(function () {
          root.animate(
            { clipPath: ['circle(0px at ' + x + 'px ' + y + 'px)', 'circle(' + end + 'px at ' + x + 'px ' + y + 'px)'] },
            { duration: 540, easing: 'ease-in-out', pseudoElement: '::view-transition-new(root)' }
          );
        }).catch(function () {});
        return;
      }
      overlayReveal(next, x, y);
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
