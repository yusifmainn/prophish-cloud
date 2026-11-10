/* SecDrill WYSIWYG redaktoru — eyni mənşəli (CSP script-src 'self'), xarici kitabxana yoxdur.
   <textarea data-editor="email|landing"> elementini vizual redaktora çevirir: toolbar,
   dəyişən əlavəsi və HTML-mənbə keçidi. Textarea DOM-da (gizli) qalır və sinxron saxlanılır,
   ona görə forma əvvəlki kimi HTML göndərir. JS sönülüdürsə, textarea adi şəkildə işləyir. */
(function () {
  var VARS = {
    email: [
      ['{{.FirstName}}', 'Ad'], ['{{.LastName}}', 'Soyad'], ['{{.Email}}', 'E-poçt'],
      ['{{.Position}}', 'Vəzifə'], ['{{.URL}}', 'Phishing linki'], ['{{.TrackingPixel}}', 'İzləmə pikseli']
    ],
    landing: [
      ['{{.FirstName}}', 'Ad'], ['{{.LastName}}', 'Soyad'], ['{{.Position}}', 'Vəzifə'],
      ['{{.SubmitURL}}', 'Forma ünvanı']
    ]
  };

  function el(tag, cls, html) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (html != null) e.innerHTML = html;
    return e;
  }
  function exec(cmd, val) {
    try { document.execCommand(cmd, false, val == null ? null : val); } catch (e) {}
  }
  function btn(cls, label, title, onClick) {
    var b = el('button', cls, label);
    b.type = 'button';
    b.title = title || '';
    b.addEventListener('mousedown', function (e) { e.preventDefault(); }); // seçimi itirmə
    b.addEventListener('click', function (e) { e.preventDefault(); onClick(); });
    return b;
  }

  function insertAtCursor(ta, text) {
    var start = ta.selectionStart == null ? ta.value.length : ta.selectionStart;
    var end = ta.selectionEnd == null ? ta.value.length : ta.selectionEnd;
    ta.value = ta.value.slice(0, start) + text + ta.value.slice(end);
    ta.selectionStart = ta.selectionEnd = start + text.length;
    ta.focus();
  }

  // Landing pages are full HTML documents (<!doctype> + <html>/<head>/<body>).
  // A contentEditable RTE silently drops the doctype and document-shell attributes
  // on read-back, corrupting the stored page that is served to live targets — so
  // landing uses a plain monospace source editor with only a variable-insert helper.
  function enhanceLanding(ta) {
    var wrap = el('div', 'rte');
    var bar = el('div', 'rte-toolbar');
    bar.appendChild(el('span', 'rte-srclabel', 'HTML mənbəyi'));
    var sel = el('select', 'rte-vars');
    sel.appendChild(el('option', null, 'Dəyişən əlavə et…'));
    var vs = VARS.landing;
    for (var i = 0; i < vs.length; i++) {
      var o = el('option');
      o.value = vs[i][0];
      o.text = vs[i][1] + '  ' + vs[i][0];
      sel.appendChild(o);
    }
    sel.addEventListener('change', function () {
      if (sel.value) { insertAtCursor(ta, sel.value); sel.selectedIndex = 0; }
    });
    bar.appendChild(sel);
    ta.parentNode.insertBefore(wrap, ta);
    wrap.appendChild(bar);
    wrap.appendChild(ta);
    ta.classList.add('rte-code');
  }

  function enhance(ta) {
    if (ta.getAttribute('data-rte') === '1') return;
    ta.setAttribute('data-rte', '1');
    ta.removeAttribute('required'); // 'required' on a soon-to-be-hidden textarea blocks submit with only a console warning
    var kind = ta.getAttribute('data-editor') === 'landing' ? 'landing' : 'email';
    if (kind === 'landing') { enhanceLanding(ta); return; }

    var wrap = el('div', 'rte');
    var bar = el('div', 'rte-toolbar');
    var area = el('div', 'rte-area');
    area.contentEditable = 'true';
    area.innerHTML = ta.value;

    function sync() { if (!srcOn) ta.value = area.innerHTML; }

    bar.appendChild(btn('rte-btn', '<b>B</b>', 'Qalın', function () { exec('bold'); sync(); }));
    bar.appendChild(btn('rte-btn', '<i>I</i>', 'Maili', function () { exec('italic'); sync(); }));
    bar.appendChild(btn('rte-btn', '<u>U</u>', 'Altxətt', function () { exec('underline'); sync(); }));
    bar.appendChild(btn('rte-btn', 'H', 'Başlıq', function () { exec('formatBlock', '<h2>'); sync(); }));
    bar.appendChild(btn('rte-btn', '¶', 'Abzas', function () { exec('formatBlock', '<p>'); sync(); }));
    bar.appendChild(btn('rte-btn', '• ≡', 'Siyahı', function () { exec('insertUnorderedList'); sync(); }));
    bar.appendChild(btn('rte-btn', '🔗', 'Link', function () {
      var url = window.prompt('Link ünvanı (phishing üçün {{.URL}} yazın):', '{{.URL}}');
      if (url) { exec('createLink', url); sync(); }
    }));

    var sel = el('select', 'rte-vars');
    sel.appendChild(el('option', null, 'Dəyişən əlavə et…'));
    var vs = VARS[kind];
    for (var i = 0; i < vs.length; i++) {
      var o = el('option');
      o.value = vs[i][0];
      o.text = vs[i][1] + '  ' + vs[i][0];
      sel.appendChild(o);
    }
    sel.addEventListener('change', function () {
      if (sel.value) { area.focus(); exec('insertText', sel.value); sync(); sel.selectedIndex = 0; }
    });
    bar.appendChild(sel);

    var srcOn = false;
    var toggle = btn('rte-btn rte-src', '&lt;/&gt;', 'HTML mənbəyi', function () {
      srcOn = !srcOn;
      if (srcOn) {
        ta.value = area.innerHTML;
        ta.style.display = ''; area.style.display = 'none'; toggle.classList.add('on');
      } else {
        area.innerHTML = ta.value;
        ta.style.display = 'none'; area.style.display = ''; toggle.classList.remove('on');
      }
    });
    bar.appendChild(toggle);

    area.addEventListener('input', sync);
    area.addEventListener('blur', sync);
    if (ta.form) ta.form.addEventListener('submit', sync);

    ta.parentNode.insertBefore(wrap, ta);
    wrap.appendChild(bar);
    wrap.appendChild(area);
    wrap.appendChild(ta);
    ta.style.display = 'none';
  }

  function init() {
    var tas = document.querySelectorAll('textarea[data-editor]');
    for (var i = 0; i < tas.length; i++) { enhance(tas[i]); }
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
  else init();
})();
