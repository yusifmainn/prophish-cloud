/* SecDrill admin — progressive enhancement served from same origin (CSP: script-src 'self').
   Applies the saved theme before body render, then wires the toggle + active nav on load. */
(function () {
  var root = document.documentElement;

  function stored() {
    try { return localStorage.getItem('secdrill-theme'); } catch (e) { return null; }
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

  var SUN = '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M2 12h2M20 12h2M5 5l1.5 1.5M17.5 17.5L19 19M19 5l-1.5 1.5M6.5 17.5L5 19"/>';
  var MOON = '<path d="M21 12.8A9 9 0 1111.2 3a7 7 0 009.8 9.8z"/>';

  function paintIcon() {
    var btn = document.getElementById('theme-toggle');
    if (!btn) return;
    var svg = btn.querySelector('svg');
    if (svg) svg.innerHTML = effective() === 'dark' ? MOON : SUN;
  }
  function wireToggle() {
    var btn = document.getElementById('theme-toggle');
    if (!btn) return;
    btn.addEventListener('click', function () {
      var next = effective() === 'dark' ? 'light' : 'dark';
      root.setAttribute('data-theme', next);
      try { localStorage.setItem('secdrill-theme', next); } catch (e) {}
      paintIcon();
    });
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

  function init() { paintIcon(); wireToggle(); markActive(); wireConfirms(); wireFlash(); wireTabs(); wireCampaignForm(); wireLivePoll(); }

  if (window.matchMedia) {
    try { window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', paintIcon); } catch (e) {}
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
