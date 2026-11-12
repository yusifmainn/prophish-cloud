/* PROPhish admin — progressive enhancement served from same origin (CSP: script-src 'self').
   Applies the saved theme before body render, then wires the toggle + active nav on load. */
(function () {
  var root = document.documentElement;

  function stored() {
    try { return localStorage.getItem('prophish-theme'); } catch (e) { return null; }
  }
  function systemDark() {
    return !!(window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches);
  }
  function effective() {
    var a = root.getAttribute('data-theme');
    if (a === 'dark' || a === 'light') return a;
    return systemDark() ? 'dark' : 'light';
  }

  // Apply saved theme immediately (this script is in <head>, before <body> renders).
  var saved = stored();
  if (saved === 'dark' || saved === 'light') root.setAttribute('data-theme', saved);

  // Theme toggle. The sun/moon icon is picked in CSS from the active theme, so
  // this only flips data-theme (+ persists it).
  function flipTheme(next) {
    root.setAttribute('data-theme', next);
    try { localStorage.setItem('prophish-theme', next); } catch (e) {}
  }
  // overlayReveal animates the switch in ANY browser (no View Transitions needed):
  // a circle of the NEW page colour grows from the control, then the theme flips
  // as it completes. A plain <div> clip-path circle, GPU-composited and crisp.
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
        { duration: 440, easing: 'cubic-bezier(.22,.61,.36,1)' }
      ).onfinish = finish;
    } catch (e) { finish(); return; }
    window.setTimeout(finish, 520);
  }
  // setTheme animates the switch as a circular reveal growing from the clicked
  // control across the page — the View Transitions API (reveals the real new
  // content) where it exists, otherwise the colour-circle overlay. It always
  // animates, in every browser.
  function setTheme(next, originEl) {
    var x = window.innerWidth - 48, y = window.innerHeight - 96;
    if (originEl && originEl.getBoundingClientRect) {
      var r = originEl.getBoundingClientRect();
      x = r.left + r.width / 2;
      y = r.top + r.height / 2;
    }
    if (typeof document.startViewTransition === 'function') {
      var end = Math.hypot(Math.max(x, window.innerWidth - x), Math.max(y, window.innerHeight - y));
      var vt = document.startViewTransition(function () { flipTheme(next); });
      vt.ready.then(function () {
        root.animate(
          { clipPath: ['circle(0px at ' + x + 'px ' + y + 'px)', 'circle(' + end + 'px at ' + x + 'px ' + y + 'px)'] },
          { duration: 440, easing: 'cubic-bezier(.22,.61,.36,1)', pseudoElement: '::view-transition-new(root)' }
        );
      }).catch(function () {});
      return;
    }
    overlayReveal(next, x, y);
  }
  function wireToggle() {
    var btns = document.querySelectorAll('#theme-toggle, [data-theme-toggle]');
    for (var i = 0; i < btns.length; i++) {
      (function (btn) {
        btn.addEventListener('click', function () {
          setTheme(effective() === 'dark' ? 'light' : 'dark', btn);
        });
      })(btns[i]);
    }
  }
  function markActive() {
    var path = location.pathname;
    var links = document.querySelectorAll('.rail nav a');
    for (var i = 0; i < links.length; i++) {
      var href = links[i].getAttribute('href');
      if (!href) continue;
      var active = href === '/' ? path === '/' : (path === href || path.indexOf(href + '/') === 0);
      links[i].classList.toggle('active', active);
    }
  }
  // Confirmation for destructive actions. Inline onsubmit/onclick handlers are
  // blocked by the Content-Security-Policy (script-src 'self'), so confirmation
  // is wired here instead: any <form data-confirm="..."> asks before submitting.
  function wireConfirms() {
    document.addEventListener('submit', function (e) {
      var f = e.target;
      if (f && f.getAttribute) {
        var msg = f.getAttribute('data-confirm');
        if (msg && !window.confirm(msg)) {
          e.preventDefault();
          e.stopPropagation();
        }
      }
    }, true);
  }

  // Auto-dismiss the one-off flash toast after a few seconds (progressive
  // enhancement; the banner is already visible without JS).
  function wireFlash() {
    var fb = document.getElementById('flashbar');
    if (!fb) return;
    setTimeout(function () {
      fb.classList.add('hide');
      setTimeout(function () { if (fb.parentNode) fb.parentNode.removeChild(fb); }, 350);
    }, 4500);
  }

  // Tabbed sections. Progressive enhancement: with no JS, all panels stay visible.
  function wireTabs() {
    var wraps = document.querySelectorAll('[data-tabs]');
    for (var w = 0; w < wraps.length; w++) {
      (function (wrap) {
        var tabs = wrap.querySelectorAll('.tabs a');
        var panels = wrap.querySelectorAll('[data-panel]');
        if (!tabs.length || !panels.length) return;
        function show(name) {
          for (var i = 0; i < panels.length; i++) {
            panels[i].style.display = panels[i].getAttribute('data-panel') === name ? '' : 'none';
          }
          for (var j = 0; j < tabs.length; j++) {
            tabs[j].classList.toggle('active', (tabs[j].getAttribute('href') || '').slice(1) === name);
          }
        }
        for (var k = 0; k < tabs.length; k++) {
          tabs[k].addEventListener('click', function (e) {
            e.preventDefault();
            show((this.getAttribute('href') || '').slice(1));
          });
        }
        var initial = (location.hash || '').slice(1);
        var hasPanel = false;
        for (var p = 0; p < panels.length; p++) {
          if (panels[p].getAttribute('data-panel') === initial) { hasPanel = true; break; }
        }
        show(hasPanel ? initial : (tabs[0].getAttribute('href') || '').slice(1));
      })(wraps[w]);
    }
  }

  // Campaign create form: filter authorizations/groups by the chosen client and
  // keep a live recipient count; disable submit when there are zero recipients.
  function wireCampaignForm() {
    var form = document.querySelector('[data-campaign-form]');
    if (!form) return;
    var client = form.querySelector('select[name="client_id"]');
    var auth = form.querySelector('select[name="authorization_id"]');
    var groups = form.querySelectorAll('[data-group]');
    var out = form.querySelector('[data-count-out]');
    var submit = form.querySelector('button[type="submit"]');
    if (!client) return;
    function apply() {
      var c = client.value;
      if (auth) {
        for (var i = 0; i < auth.options.length; i++) {
          var o = auth.options[i], dc = o.getAttribute('data-client');
          var hide = !!(c && dc && dc !== c);
          o.hidden = hide; o.disabled = hide;
          if (hide && o.selected) auth.value = '';
        }
      }
      var total = 0;
      for (var j = 0; j < groups.length; j++) {
        var g = groups[j], gc = g.getAttribute('data-client');
        var show = !c || gc === c;
        g.style.display = show ? '' : 'none';
        var cb = g.querySelector('input[type="checkbox"]');
        if (cb) {
          if (!show && cb.checked) cb.checked = false;
          if (cb.checked) total += (parseInt(g.getAttribute('data-count'), 10) || 0);
        }
      }
      // When a client is chosen but it has no authorization or no groups, the
      // operator would otherwise face an empty İcazə select and an empty target
      // area with no explanation — reveal a pre-rendered hint pointing at /clients.
      var authCount = 0;
      if (auth) {
        for (var a = 0; a < auth.options.length; a++) {
          if (auth.options[a].value && !auth.options[a].hidden) authCount++;
        }
      }
      var groupVisible = 0;
      for (var gv = 0; gv < groups.length; gv++) {
        if (groups[gv].style.display !== 'none') groupVisible++;
      }
      var hint = form.querySelector('[data-empty-hint]');
      if (hint) hint.style.display = (c && (authCount === 0 || groupVisible === 0)) ? '' : 'none';
      if (out) out.textContent = total;
      if (submit) submit.disabled = (total === 0 || !client.value);
    }
    client.addEventListener('change', apply);
    for (var k = 0; k < groups.length; k++) {
      var cb2 = groups[k].querySelector('input[type="checkbox"]');
      if (cb2) cb2.addEventListener('change', apply);
    }
    apply();
    // Listener-location radios: the custom-URL field is shown only for "custom".
    var customBox = form.querySelector('[data-custom-url]');
    var radios = form.querySelectorAll('input[name="listener_mode"]');
    function applyMode() {
      if (!customBox) return;
      var m = '';
      for (var i = 0; i < radios.length; i++) if (radios[i].checked) m = radios[i].value;
      customBox.style.display = m === 'custom' ? '' : 'none';
    }
    for (var r = 0; r < radios.length; r++) radios[r].addEventListener('change', applyMode);
    applyMode();
  }

  // Live progress for a running campaign: poll the page itself (same origin) and
  // swap just the funnel + risk block and the sent/total counters in place, so the
  // operator's scroll position is preserved (unlike a full meta-refresh). When the
  // campaign finishes, the fresh document no longer has the running banner → reload
  // once to show the final state.
  function wireLivePoll() {
    var banner = document.querySelector('[data-live-poll]');
    if (!banner || !window.fetch || !window.DOMParser) return;
    var INTERVAL = 12000;
    var timer = null;
    function tick() {
      fetch(location.href, { credentials: 'same-origin', headers: { 'X-Requested-With': 'fetch' } })
        .then(function (res) { return res.ok ? res.text() : null; })
        .then(function (html) {
          if (!html) return;
          var doc = new DOMParser().parseFromString(html, 'text/html');
          if (!doc.querySelector('[data-live-poll]')) { location.reload(); return; }
          var curGrid = document.querySelector('.rgrid'), newGrid = doc.querySelector('.rgrid');
          if (curGrid && newGrid) curGrid.innerHTML = newGrid.innerHTML;
          var ns = doc.querySelector('[data-live-sent]'), bs = banner.querySelector('[data-live-sent]');
          if (ns && bs) bs.textContent = ns.textContent;
          var nt = doc.querySelector('[data-live-total]'), bt = banner.querySelector('[data-live-total]');
          if (nt && bt) bt.textContent = nt.textContent;
        })
        .catch(function () {});
    }
    function start() { if (!timer) timer = setInterval(tick, INTERVAL); }
    function stop() { if (timer) { clearInterval(timer); timer = null; } }
    var toggle = banner.querySelector('[data-live-toggle]');
    if (toggle) {
      toggle.addEventListener('click', function () {
        if (timer) { stop(); toggle.textContent = 'Davam et'; }
        else { start(); toggle.textContent = 'Dayandır'; }
      });
    }
    start();
  }

  // Select-all + per-row checkboxes for the dashboard campaign table. Keeps the
  // header checkbox in sync (checked / indeterminate) and enables the bulk
  // action button only when at least one row is selected.
  function wireBulkSelect() {
    var all = document.querySelector('[data-check-all]');
    if (!all) return;
    var form = all.closest('form');
    if (!form) return;
    function rows() { return Array.prototype.slice.call(form.querySelectorAll('[data-row-check]')); }
    var submit = form.querySelector('[data-bulk-submit]');
    var count = form.querySelector('[data-bulk-count]');
    function sync() {
      var cbs = rows();
      var checked = cbs.filter(function (c) { return c.checked; });
      all.checked = cbs.length > 0 && checked.length === cbs.length;
      all.indeterminate = checked.length > 0 && checked.length < cbs.length;
      if (submit) submit.disabled = checked.length === 0;
      if (count) count.textContent = checked.length ? ' (' + checked.length + ')' : '';
    }
    all.addEventListener('change', function () {
      rows().forEach(function (c) { c.checked = all.checked; });
      sync();
    });
    form.addEventListener('change', function (e) {
      if (e.target && e.target.hasAttribute && e.target.hasAttribute('data-row-check')) sync();
    });
    sync();
  }

  function init() { wireToggle(); markActive(); wireConfirms(); wireFlash(); wireTabs(); wireCampaignForm(); wireLivePoll(); wireBulkSelect(); }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
