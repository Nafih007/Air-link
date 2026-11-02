(function () {
  function go() {
    return window.go && window.go.main && window.go.main.App ? window.go.main.App : null;
  }

  function apply(s) {
    s = s || {};
    var status = document.getElementById("status");
    var qr = document.getElementById("qr");
    var url = document.getElementById("url");
    var pinbox = document.getElementById("pinbox");
    var pin = document.getElementById("pin");
    var box = document.getElementById("devices");
    var list = document.getElementById("device-list");
    var itemsEl = document.getElementById("items");
    if (status) status.textContent = s.status || "Scan with your phone camera";
    if (qr && s.qrDataURL) qr.src = s.qrDataURL;
    if (url) url.textContent = s.pageURL || s.listenAddr || "";
    var cd = document.getElementById("countdown");
    if (cd) {
      var left = s.joinLeftSec || 0;
      cd.textContent = left > 0 ? "This QR expires in " + left + "s" : "Refreshing QR…";
    }
    var auto = document.getElementById("autoallow");
    if (auto && document.activeElement !== auto) auto.checked = !!s.autoAllow;
    if (pinbox && pin) {
      pinbox.hidden = !s.pin;
      if (s.pin) pin.textContent = s.pin;
    }
    var devices = s.devices || [];
    if (box && list) {
      box.hidden = devices.length === 0;
      list.innerHTML = "";
      devices.forEach(function (d) {
        var row = document.createElement("div");
        row.className = "dev-row";
        var lab = document.createElement("span");
        var extra = d.pending ? " — waiting Allow" : d.denied ? " (denied)" : d.kicked ? " (kicked)" : "";
        lab.textContent = "New device: " + (d.browser || "Browser") + (d.ip ? " · " + d.ip : "") + extra;
        if (!d.pending) lab.textContent = (d.browser || "Browser") + (d.ip ? " · " + d.ip : "") + extra;
        row.appendChild(lab);
        var acts = document.createElement("div");
        acts.className = "item-actions";
        if (d.pending) {
          var ok = document.createElement("button");
          ok.type = "button";
          ok.className = "tiny";
          ok.textContent = "Allow";
          ok.onclick = function () { var a = go(); if (a) a.AllowDevice(d.id); };
          var no = document.createElement("button");
          no.type = "button";
          no.className = "tiny danger";
          no.textContent = "Deny";
          no.onclick = function () { var a = go(); if (a) a.DenyDevice(d.id); };
          acts.appendChild(ok);
          acts.appendChild(no);
        } else if (!d.kicked && !d.denied) {
          var b = document.createElement("button");
          b.type = "button";
          b.className = "tiny danger";
          b.textContent = "Kick";
          b.onclick = function () { var a = go(); if (a) a.KickDevice(d.id); };
          acts.appendChild(b);
        }
        row.appendChild(acts);
        list.appendChild(row);
      });
    }
    if (!itemsEl) return;
    itemsEl.innerHTML = "";
    var items = s.items || [];
    var empty = document.getElementById("timeline-empty");
    if (empty) empty.hidden = items.length > 0;
    items.slice().reverse().forEach(function (it) {
      var row = document.createElement("div");
      row.className = "item";
      var left = document.createElement("div");
      var title = document.createElement("div");
      title.textContent = (it.direction === "in" ? "← " : "→ ") + (it.name || "item");
      var meta = document.createElement("div");
      meta.className = "meta " + (it.status === "done" || it.status === "ready" ? "ok" : it.status === "pending" ? "wait" : "fail");
      meta.textContent = (it.status || "") + (it.detail ? " · " + it.detail : "");
      left.appendChild(title);
      left.appendChild(meta);
      row.appendChild(left);
      if (it.status === "pending" && it.direction === "in") {
        var acts = document.createElement("div");
        acts.className = "item-actions";
        var ok = document.createElement("button");
        ok.type = "button";
        ok.className = "tiny";
        ok.textContent = "Accept";
        ok.onclick = function () { var a = go(); if (a) a.AcceptUpload(it.id); };
        var no = document.createElement("button");
        no.type = "button";
        no.className = "tiny danger";
        no.textContent = "Decline";
        no.onclick = function () { var a = go(); if (a) a.DeclineUpload(it.id); };
        acts.appendChild(ok);
        acts.appendChild(no);
        row.appendChild(acts);
      }
      itemsEl.appendChild(row);
    });
  }

  function refresh() {
    var a = go();
    if (!a) return;
    a.Snapshot().then(apply);
  }

  document.getElementById("copy").addEventListener("click", function () {
    var a = go();
    if (a) a.CopyPageURL();
  });
  document.getElementById("send").addEventListener("click", function () {
    var a = go();
    if (a) a.PickFiles().then(refresh);
  });
  document.getElementById("folder").addEventListener("click", function () {
    var a = go();
    if (a) a.OpenReceivedFolder();
  });
  document.getElementById("end").addEventListener("click", function () {
    var a = go();
    if (a) a.EndSession().then(refresh);
  });
  document.getElementById("refreshqr").addEventListener("click", function () {
    var a = go();
    if (a) a.RefreshQR().then(refresh);
  });
  document.getElementById("autoallow").addEventListener("change", function (e) {
    var a = go();
    if (a) a.SetAutoAllow(e.target.checked);
  });

  function applyMenu(st) {
    st = st || {};
    var lab = document.getElementById("menu-label");
    var det = document.getElementById("menu-detail");
    var note = document.getElementById("menu-win11");
    var inst = document.getElementById("menu-install");
    if (lab) lab.textContent = st.label || "Right-click menu: …";
    if (det) det.textContent = st.detail || (st.sendToOK ? "Also available under Send to." : "");
    if (note) note.textContent = st.win11Note || "";
    if (inst) inst.textContent = st.installed && !st.needsRepair ? "Installed ✓" : "Install";
  }

  function refreshMenu() {
    var a = go();
    if (!a || !a.ContextMenu) return;
    a.ContextMenu().then(applyMenu);
  }

  function afterMenu(fn) {
    return function () {
      var a = go();
      if (!a) return;
      fn(a).then(function (err) {
        if (err) applyMenu({ label: "Right-click menu: Error", detail: String(err) });
        refreshMenu();
      });
    };
  }

  document.getElementById("menu-install").addEventListener("click", afterMenu(function (a) { return a.InstallMenu(); }));
  document.getElementById("menu-repair").addEventListener("click", afterMenu(function (a) { return a.RepairMenu(); }));
  document.getElementById("menu-remove").addEventListener("click", afterMenu(function (a) { return a.UninstallMenu(); }));

  function waitBindings() {
    if (go()) {
      refresh();
      refreshMenu();
      if (window.runtime && window.runtime.EventsOn) {
        window.runtime.EventsOn("room:update", apply);
      }
      return;
    }
    setTimeout(waitBindings, 50);
  }
  waitBindings();
})();
