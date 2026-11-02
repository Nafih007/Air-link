(function () {
  "use strict";

  function detectPlatform(ua) {
    ua = ua || "";
    if (/iPhone|iPad|iPod/i.test(ua) || (navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1)) {
      return "ios";
    }
    if (/Android/i.test(ua)) {
      return "android";
    }
    return "other";
  }

  function canUseWebShare() {
    return window.isSecureContext === true &&
      typeof navigator.share === "function" &&
      typeof navigator.canShare === "function";
  }

  function formatSize(n) {
    if (n < 1024) return n + " bytes";
    if (n < 1024 * 1024) return (n / 1024).toFixed(1) + " KB";
    if (n < 1024 * 1024 * 1024) return (n / (1024 * 1024)).toFixed(1) + " MB";
    return (n / (1024 * 1024 * 1024)).toFixed(2) + " GB";
  }

  function showError(msg) {
    var el = document.getElementById("errors");
    if (!el) return;
    var line = "[" + new Date().toLocaleTimeString() + "] " + msg;
    el.textContent = el.textContent ? el.textContent + "\n" + line : line;
    el.hidden = false;
  }

  function showFallback(reason) {
    var box = document.getElementById("fallback");
    var why = document.getElementById("fallback-reason");
    if (why) why.textContent = reason || "";
    if (box) box.hidden = false;
  }

  async function shareFile(url, filename, mime) {
    try {
      if (!canUseWebShare()) {
        showFallback("Share is unavailable (needs HTTPS). Use Download or long-press instead.");
        return;
      }
      var res = await fetch(url);
      if (!res.ok) throw new Error("HTTP " + res.status);
      var blob = await res.blob();
      var file = new File([blob], filename, { type: mime || blob.type || "application/octet-stream" });
      var payload = { files: [file], title: filename };
      if (!navigator.canShare(payload)) {
        showFallback("This file cannot be shared from the browser. Use Download or long-press.");
        showError("canShare returned false for " + filename);
        return;
      }
      await navigator.share(payload);
    } catch (err) {
      var msg = err && err.message ? err.message : String(err);
      showError("Share failed: " + msg);
      showFallback("Share failed. Use Download or long-press → Save to Photos.");
    }
  }

  function selectedIndexes(gallery, files) {
    var selected = [];
    if (!gallery) return selected;
    gallery.querySelectorAll("input[type=checkbox]").forEach(function (cb) {
      if (cb.checked) {
        var i = Number(cb.dataset.index);
        if (!isNaN(i) && files[i]) selected.push(i);
      }
    });
    return selected;
  }

  function zipURL(indexes) {
    if (!indexes || !indexes.length) return "download";
    return "download?files=" + indexes.join(",");
  }

  function triggerDownload(href, filename) {
    var a = document.createElement("a");
    a.href = href;
    a.setAttribute("download", filename || "download");
    a.style.display = "none";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  }

  function setUploadStatus(msg, kind) {
    var el = document.getElementById("upload-status");
    if (!el) return;
    el.hidden = !msg;
    el.textContent = msg || "";
    el.className = kind || "";
  }

  function resumeKey(files, text) {
    var parts = [];
    for (var i = 0; i < files.length; i++) {
      parts.push(files[i].name + ":" + files[i].size);
    }
    parts.push("t:" + (text || "").length);
    return "stp-up-" + parts.join("|");
  }

  function wireSendToPC() {
    var btn = document.getElementById("upload-send");
    var fileInput = document.getElementById("upload-files");
    var textInput = document.getElementById("upload-text");
    if (!btn || btn.dataset.wired === "1") return;
    btn.dataset.wired = "1";
    btn.addEventListener("click", function () {
      var files = fileInput && fileInput.files ? Array.prototype.slice.call(fileInput.files) : [];
      var text = textInput ? String(textInput.value || "").trim() : "";
      if (!files.length && !text) {
        setUploadStatus("Pick a file or paste text first.", "wait");
        return;
      }
      btn.disabled = true;
      setUploadStatus("Sending…", "wait");
      if (!files.length) {
        var fd = new FormData();
        fd.append("text", text);
        fetch("upload", { method: "POST", body: fd })
          .then(function (r) { return r.json().then(function (j) { return { ok: r.ok, body: j }; }); })
          .then(function (res) { return handleUploadResult(res.body, btn, fileInput, textInput); })
          .catch(function (err) {
            setUploadStatus("Upload failed: " + (err && err.message ? err.message : String(err)), "wait");
            btn.disabled = false;
          });
        return;
      }
      sendChunked(files, text, btn, fileInput, textInput);
    });
  }

  function handleUploadResult(body, btn, fileInput, textInput) {
    body = body || {};
    if (body.status === "saved") {
      var msg = body.message || "Saved on PC";
      if (body.saved && body.saved.length) {
        msg += ": " + body.saved.join(", ");
      }
      setUploadStatus(msg, "ok");
      if (fileInput) fileInput.value = "";
      if (textInput) textInput.value = "";
      btn.disabled = false;
      return;
    }
    if (body.status === "pending") {
      setUploadStatus(body.message || "Waiting for PC approval…", "wait");
      return pollUploadStatus(body.id, btn, fileInput, textInput);
    }
    setUploadStatus(body.message || "Upload failed", "wait");
    btn.disabled = false;
  }

  function sendChunked(files, text, btn, fileInput, textInput) {
    var key = resumeKey(files, text);
    var savedId = null;
    try { savedId = sessionStorage.getItem(key); } catch (e) {}

    var start = savedId
      ? Promise.resolve({ id: savedId, chunkSize: 1048576 })
      : fetch("upload-init", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            text: text || "",
            files: files.map(function (f) { return { name: f.name, size: f.size }; })
          })
        }).then(function (r) {
          if (!r.ok) throw new Error("init HTTP " + r.status);
          return r.json();
        });

    start.then(function (init) {
      var id = init.id;
      var chunkSize = init.chunkSize || 1048576;
      try { sessionStorage.setItem(key, id); } catch (e) {}

      return fetch("upload-status?id=" + encodeURIComponent(id))
        .then(function (r) { return r.json(); })
        .then(function (st) {
          if (st && st.status === "unknown") {
            try { sessionStorage.removeItem(key); } catch (e) {}
            throw new Error("Previous upload expired — tap Send again");
          }
          if (st && (st.status === "saved" || st.status === "pending" || st.status === "declined")) {
            try { sessionStorage.removeItem(key); } catch (e) {}
            return handleUploadResult(st, btn, fileInput, textInput);
          }
          return uploadAllChunks(id, chunkSize, files, st).then(function () {
            return fetch("upload-finish", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ id: id })
            }).then(function (r) {
              return r.json().then(function (j) {
                if (!r.ok && r.status !== 202) throw new Error((j && j.message) || "finish failed");
                try { sessionStorage.removeItem(key); } catch (e) {}
                return handleUploadResult(j, btn, fileInput, textInput);
              });
            });
          });
        });
    }).catch(function (err) {
      setUploadStatus("Upload failed: " + (err && err.message ? err.message : String(err)), "wait");
      showError("Upload failed: " + (err && err.message ? err.message : String(err)));
      btn.disabled = false;
    });
  }

  function uploadAllChunks(id, chunkSize, files, status) {
    var received = {};
    if (status && status.files) {
      status.files.forEach(function (f) {
        received[f.index] = f.received || 0;
      });
    }
    var chain = Promise.resolve();
    files.forEach(function (file, fileIndex) {
      var totalChunks = Math.max(1, Math.ceil(file.size / chunkSize) || 1);
      var startChunk = received[fileIndex] || 0;
      for (var c = startChunk; c < totalChunks; c++) {
        (function (fi, ci, total) {
          chain = chain.then(function () {
            var start = ci * chunkSize;
            var end = Math.min(file.size, start + chunkSize);
            var blob = file.slice(start, end);
            setUploadStatus("Sending " + file.name + " (" + (ci + 1) + "/" + total + ")…", "wait");
            return putChunk(id, fi, ci, blob).catch(function () {
              return delay(500).then(function () { return putChunk(id, fi, ci, blob); });
            }).then(function (prog) {
              if (prog && prog.total > 0) {
                var pct = Math.floor((prog.got / prog.total) * 100);
                setUploadStatus("Sending… " + pct + "% (" + formatSize(prog.got) + " / " + formatSize(prog.total) + ")", "wait");
              }
            });
          });
        })(fileIndex, c, totalChunks);
      }
    });
    return chain;
  }

  function putChunk(id, fileIndex, chunkIndex, blob) {
    return fetch("upload-chunk", {
      method: "PUT",
      headers: {
        "Content-Type": "application/octet-stream",
        "X-Upload-Id": id,
        "X-File-Index": String(fileIndex),
        "X-Chunk-Index": String(chunkIndex)
      },
      body: blob
    }).then(function (r) {
      if (!r.ok) throw new Error("chunk HTTP " + r.status);
      return r.json();
    });
  }

  function delay(ms) {
    return new Promise(function (resolve) { setTimeout(resolve, ms); });
  }

  function pollUploadStatus(id, btn, fileInput, textInput) {
    if (!id) {
      if (btn) btn.disabled = false;
      return;
    }
    var tries = 0;
    var timer = setInterval(function () {
      tries++;
      fetch("upload-status?id=" + encodeURIComponent(id))
        .then(function (r) { return r.json(); })
        .then(function (body) {
          if (!body || body.status === "pending" || body.status === "receiving") {
            if (body && body.status === "receiving" && body.total > 0) {
              var pct = Math.floor((body.got / body.total) * 100);
              setUploadStatus("Sending… " + pct + "%", "wait");
            }
            if (tries > 600) {
              clearInterval(timer);
              setUploadStatus("Still waiting for PC approval…", "wait");
              if (btn) btn.disabled = false;
            }
            return;
          }
          clearInterval(timer);
          if (body.status === "saved") {
            var msg = body.message || "Saved on PC";
            if (body.saved && body.saved.length) {
              msg += ": " + body.saved.join(", ");
            }
            setUploadStatus(msg, "ok");
            if (fileInput) fileInput.value = "";
            if (textInput) textInput.value = "";
          } else if (body.status === "declined") {
            setUploadStatus("Declined on PC", "wait");
          } else {
            setUploadStatus(body.message || "Upload ended", "wait");
          }
          if (btn) btn.disabled = false;
        })
        .catch(function () {
          /* keep polling */
        });
    }, 1000);
  }

  function wirePair(meta) {
    var box = document.getElementById("pair-box");
    var btn = document.getElementById("pair-btn");
    var steps = document.getElementById("pair-steps");
    if (!box || !btn) return;
    if (!meta || !meta.canPair) {
      box.hidden = true;
      return;
    }
    box.hidden = false;
    function showSteps(apiUrl, token) {
      if (!steps) return;
      steps.hidden = false;
      steps.textContent =
        "Shortcut: Share Sheet → Send to PC\n" +
        "Get Contents of URL\n" +
        "URL: " + apiUrl + "\n" +
        "Method: POST\n" +
        "Headers: Authorization = Bearer " + token + "\n" +
        "Request body: Form\n" +
        "  files = Shortcut Input (file)\n" +
        "  text = Shortcut Input (text, optional)\n" +
        "Add to Home Screen: open " + (meta.homeUrl || "/home") + " then Share → Add to Home Screen.\n" +
        "That page remembers this token and uploads without a new QR.\n" +
        "Full steps: see docs/IPHONE_SHORTCUT.md on the PC.";
    }
    var saved = null;
    try { saved = localStorage.getItem("stp-device-token"); } catch (e) {}
    if (saved && meta.apiUrl) showSteps(meta.apiUrl, saved);
    if (btn.dataset.wired === "1") return;
    btn.dataset.wired = "1";
    btn.addEventListener("click", function () {
      btn.disabled = true;
      fetch("pair", { method: "POST", credentials: "same-origin" })
        .then(function (r) {
          if (!r.ok) throw new Error("pair HTTP " + r.status);
          return r.json();
        })
        .then(function (j) {
          try {
            localStorage.setItem("stp-device-token", j.deviceToken);
            if (meta.apiUrl) localStorage.setItem("stp-api-url", meta.apiUrl);
          } catch (e) {}
          showSteps(meta.apiUrl, j.deviceToken);
          btn.disabled = false;
        })
        .catch(function (err) {
          showError("Pair failed: " + (err && err.message ? err.message : String(err)));
          btn.disabled = false;
        });
    });
  }

  function render(meta) {
    var platform = detectPlatform(navigator.userAgent || "");
    document.body.setAttribute("data-platform", platform);

    var nameEl = document.getElementById("name");
    var sizeEl = document.getElementById("size");
    var platformEl = document.getElementById("platform");
    if (nameEl) nameEl.textContent = meta.displayName;
    if (sizeEl) sizeEl.textContent = formatSize(meta.size);
    document.title = meta.displayName + " · Air-Link";
    if (platformEl) {
      platformEl.textContent = platform === "ios" ? "iPhone" : platform === "android" ? "Android" : "Browser";
    }

    wirePair(meta);

    var media = document.getElementById("media");
    var actions = document.getElementById("actions");
    var gallery = document.getElementById("gallery");
    gallery.hidden = true;
    gallery.innerHTML = "";
    actions.innerHTML = "";

    function addBtn(text, className, onClick) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = className || "btn";
      b.textContent = text;
      b.addEventListener("click", onClick);
      actions.appendChild(b);
      return b;
    }
    function addLink(text, href, downloadName, className) {
      var a = document.createElement("a");
      a.className = className || "btn secondary";
      a.href = href;
      a.textContent = text;
      if (downloadName) a.setAttribute("download", downloadName);
      actions.appendChild(a);
      return a;
    }
    function addHint(text) {
      var hint = document.createElement("p");
      hint.className = "hint";
      hint.textContent = text;
      actions.appendChild(hint);
    }

    if ((meta.mode === "receive" || meta.mode === "room") && !(meta.files && meta.files.length)) {
      media.innerHTML = "";
      if (meta.mode === "room") {
        if (nameEl) nameEl.textContent = "Waiting for the PC to send…";
        if (sizeEl) sizeEl.textContent = "You can also send files to the PC below";
        document.title = "Air-Link";
        addHint("This page stays open. Send to PC, or wait for files from the PC.");
      } else {
        if (nameEl) nameEl.textContent = "Send files to the PC";
        if (sizeEl) sizeEl.textContent = "No download — upload only";
        document.title = "Air-Link";
        addHint("Use Send to PC below to pick photos, files, or paste text.");
      }
      wireSendToPC();
      return;
    }

    if (meta.mode === "text") {
      media.innerHTML = "";
      var pre = document.createElement("pre");
      pre.id = "text-body";
      pre.style.cssText = "white-space:pre-wrap;word-break:break-word;background:#0b0f14;padding:0.85rem;border-radius:10px;max-height:50vh;overflow:auto;font-size:0.95rem;";
      pre.textContent = meta.text || "";
      media.appendChild(pre);
      addBtn("Copy", "btn", function () {
        var t = meta.text || "";
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(t).then(function () {
            showError("Copied to clipboard");
          }).catch(function (err) {
            showError("Copy failed: " + err);
          });
        } else {
          var range = document.createRange();
          range.selectNodeContents(pre);
          var sel = window.getSelection();
          sel.removeAllRanges();
          sel.addRange(range);
          try {
            document.execCommand("copy");
            showError("Copied to clipboard");
          } catch (e) {
            showError("Copy failed — select the text manually");
          }
        }
      });
      addLink("Download .txt", "download", meta.downloadName || "text.txt", "btn secondary");
      wireSendToPC();
      return;
    }

    var shareOk = canUseWebShare();
    if (!shareOk) {
      showFallback(
        window.isSecureContext
          ? "Share is not available in this browser."
          : "This page is not a secure context (HTTP on LAN). Share to Photos needs HTTPS; use Download or long-press."
      );
    }

    var files = meta.files || [];
    var allImages = files.length > 1 && files.every(function (f) { return f.kind === "image"; });

    if (files.length === 1 && files[0].kind === "image") {
      media.innerHTML = "";
      var img = document.createElement("img");
      img.src = files[0].url;
      img.alt = files[0].name;
      media.appendChild(img);
    } else if (files.length === 1 && files[0].kind === "video") {
      media.innerHTML = "";
      var video = document.createElement("video");
      video.src = files[0].url;
      video.controls = true;
      video.setAttribute("playsinline", "");
      media.appendChild(video);
    } else {
      media.innerHTML = "";
    }

    if (allImages) {
      gallery.hidden = false;
      gallery.innerHTML = "";
      files.forEach(function (f, i) {
        var label = document.createElement("label");
        label.className = "thumb";
        var cb = document.createElement("input");
        cb.type = "checkbox";
        cb.checked = true;
        cb.dataset.index = String(i);
        var im = document.createElement("img");
        im.src = f.url;
        im.alt = f.name;
        label.appendChild(cb);
        label.appendChild(im);
        gallery.appendChild(label);
      });
    } else {
      gallery.hidden = true;
      gallery.innerHTML = "";
    }

    // Multi-image gallery: checkboxes + zip download are one flow.
    if (allImages) {
      addBtn("Download selected (zip)", "btn", function () {
        var idx = selectedIndexes(gallery, files);
        if (!idx.length) {
          showError("Select at least one image");
          return;
        }
        triggerDownload(zipURL(idx), idx.length === files.length ? meta.downloadName : "selected.zip");
      });
      addLink("Download all (zip)", "download", meta.downloadName, "btn secondary");
      if (shareOk) {
        addBtn("Share selected", "btn secondary", function () {
          var idx = selectedIndexes(gallery, files);
          if (!idx.length) {
            showError("Select at least one image");
            return;
          }
          (async function () {
            for (var i = 0; i < idx.length; i++) {
              var f = files[idx[i]];
              await shareFile(f.url, f.name, f.mime);
            }
          })();
        });
      }
      addHint("Tick images, then Download selected — or long-press a thumbnail → Save to Photos.");
      wireSendToPC();
      return;
    }

    // Single file / non-gallery
    if (platform === "ios") {
      if (shareOk && files.length === 1 && (files[0].kind === "image" || files[0].kind === "video")) {
        addBtn("Save to Photos", "btn", function () {
          shareFile(files[0].url, files[0].name, files[0].mime);
        });
      }
      addLink("Download", "download", meta.downloadName, "btn");
      if (files.length === 1 && files[0].kind === "image") {
        addHint("Or long-press the image → Save to Photos");
      } else if (files.length === 1 && files[0].kind === "video") {
        addHint("Or long-press the video → Share → Save to Photos (if offered). Else Download, then Files → Save Video.");
      } else {
        addHint("Download opens Files. From Files you can save media to Photos.");
      }
    } else if (platform === "android") {
      addLink("Download", "download", meta.downloadName, "btn");
      if (shareOk && files.length === 1) {
        addBtn("Share", "btn secondary", function () {
          shareFile(files[0].url, files[0].name, files[0].mime);
        });
      }
    } else {
      addLink("Download", "download", meta.downloadName, "btn");
    }
    wireSendToPC();
  }

  window.AirLink = {
    detectPlatform: detectPlatform,
    canUseWebShare: canUseWebShare,
    formatSize: formatSize,
    zipURL: zipURL,
    render: render,
    showError: showError,
    showFallback: showFallback
  };

  function showPinGate(msg) {
    var gate = document.getElementById("pin-gate");
    var err = document.getElementById("pin-err");
    if (gate) gate.hidden = false;
    if (err) err.textContent = msg || "";
    var go = document.getElementById("pin-go");
    var input = document.getElementById("pin-input");
    if (go && go.dataset.wired !== "1") {
      go.dataset.wired = "1";
      go.addEventListener("click", submitPin);
      if (input) input.addEventListener("keydown", function (e) {
        if (e.key === "Enter") submitPin();
      });
    }
  }

  function submitPin() {
    var input = document.getElementById("pin-input");
    var pin = input ? String(input.value || "").trim() : "";
    fetch("pin", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ pin: pin })
    }).then(function (r) { return r.json().then(function (j) { return { status: r.status, body: j }; }); })
      .then(function (res) {
        if (res.body && res.body.ok) {
          var gate = document.getElementById("pin-gate");
          if (gate) gate.hidden = true;
          loadMeta();
          return;
        }
        if (res.body && res.body.error === "locked") {
          showPinGate("Too many attempts. This session is locked.");
          return;
        }
        var left = res.body && typeof res.body.left === "number" ? res.body.left : "?";
        showPinGate("Wrong PIN. Tries left: " + left);
      })
      .catch(function (err) {
        showPinGate("PIN failed: " + (err && err.message ? err.message : String(err)));
      });
  }

  function loadMeta() {
    fetch("meta", { credentials: "same-origin" })
      .then(function (r) {
        if (r.status === 401) {
          showPinGate("Enter the PIN shown on the PC.");
          return null;
        }
        if (r.status === 403) {
          return r.text().then(function (t) {
            var j = {};
            try { j = JSON.parse(t); } catch (e) {}
            if (j && j.error === "locked") {
              showPinGate("Too many attempts. This session is locked.");
              return null;
            }
            if (j && j.error === "pending") {
              var nameEl = document.getElementById("name");
              var sizeEl = document.getElementById("size");
              if (nameEl) nameEl.textContent = "Waiting for the PC to Allow this device";
              if (sizeEl) sizeEl.textContent = j.message || "";
              if (!window._stpPoll) window._stpPoll = setInterval(loadMeta, 1500);
              return null;
            }
            if (j && j.error === "denied") {
              showError(j.message || "The PC denied this device");
              return null;
            }
            throw new Error("Access denied");
          });
        }
        if (!r.ok) throw new Error("meta HTTP " + r.status);
        return r.json();
      })
      .then(function (meta) {
        if (!meta) return;
        var key = (meta.mode || "") + "|" + ((meta.files && meta.files.length) || 0) + "|" + (meta.size || 0) + "|" + (meta.displayName || "");
        if (key === window._stpMetaKey) return;
        window._stpMetaKey = key;
        render(meta);
        if (meta.room || meta.mode === "room") {
          wireSendToPC();
          if (!window._stpPoll) {
            window._stpPoll = setInterval(loadMeta, 2000);
          }
        }
      })
      .catch(function (err) {
        showError("Failed to load file list: " + (err && err.message ? err.message : String(err)));
        showFallback("Could not load the page data. Try refreshing.");
      });
  }

  function bootPairedHome() {
    var token = "";
    var apiUrl = "/api/upload";
    try {
      token = localStorage.getItem("stp-device-token") || "";
      apiUrl = localStorage.getItem("stp-api-url") || apiUrl;
    } catch (e) {}
    var gate = document.getElementById("pin-gate");
    if (gate) gate.hidden = true;
    var pairBox = document.getElementById("pair-box");
    if (pairBox) pairBox.hidden = true;
    document.title = "Air-Link";
    var nameEl = document.getElementById("name");
    var sizeEl = document.getElementById("size");
    if (!token) {
      if (nameEl) nameEl.textContent = "Not paired";
      if (sizeEl) sizeEl.textContent = "Scan the PC QR, tap Pair this phone, then add /home to the Home Screen.";
      return;
    }
    if (nameEl) nameEl.textContent = "Paired phone";
    if (sizeEl) sizeEl.textContent = "Files go to the PC receive folder — joining room if open…";
    fetch("/rejoin", { method: "POST", headers: { "Authorization": "Bearer " + token } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (j) {
        if (j && j.page) location.replace(j.page);
      })
      .catch(function () {});
    var btn = document.getElementById("upload-send");
    var fileInput = document.getElementById("upload-files");
    var textInput = document.getElementById("upload-text");
    if (!btn || btn.dataset.wired === "1") return;
    btn.dataset.wired = "1";
    btn.addEventListener("click", function () {
      var files = fileInput && fileInput.files ? Array.prototype.slice.call(fileInput.files) : [];
      var text = textInput ? String(textInput.value || "").trim() : "";
      if (!files.length && !text) {
        setUploadStatus("Pick a file or paste text first.", "wait");
        return;
      }
      btn.disabled = true;
      setUploadStatus("Sending…", "wait");
      var fd = new FormData();
      if (text) fd.append("text", text);
      files.forEach(function (f) { fd.append("files", f, f.name); });
      fetch(apiUrl, {
        method: "POST",
        headers: { "Authorization": "Bearer " + token },
        body: fd
      }).then(function (r) {
        return r.json().then(function (j) { return { status: r.status, body: j }; });
      }).then(function (res) {
        if (res.status === 401) {
          setUploadStatus("Token rejected — pair again from the QR.", "wait");
          btn.disabled = false;
          return;
        }
        handleUploadResult(res.body, btn, fileInput, textInput);
      }).catch(function (err) {
        setUploadStatus("Upload failed: " + (err && err.message ? err.message : String(err)), "wait");
        btn.disabled = false;
      });
    });
  }

  function bootJoin() {
    var nameEl = document.getElementById("name");
    var sizeEl = document.getElementById("size");
    if (nameEl) nameEl.textContent = "Joining…";
    fetch(location.pathname.replace(/\/?$/, "/"), {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: "{}"
    }).then(function (r) {
      return r.json().then(function (j) { return { status: r.status, body: j }; });
    }).then(function (res) {
      var j = res.body || {};
      if (j.deviceToken && j.page) {
        try { localStorage.setItem("stp-room-token", j.deviceToken); } catch (e) {}
        location.replace(j.page);
        return;
      }
      var msg = j.message || "Could not join. Scan the new QR on the PC.";
      if (nameEl) nameEl.textContent = msg;
      if (sizeEl) sizeEl.textContent = "";
      showError(msg);
    }).catch(function (err) {
      showError("Join failed: " + (err && err.message ? err.message : String(err)));
    });
  }

  var joinMatch = location.pathname.match(/^\/j\/([0-9a-f]{32})\/?$/);
  var roomMatch = location.pathname.match(/^\/t\/([0-9a-f]{64})\//);
  if (joinMatch) {
    bootJoin();
  } else if (/^\/home\/?$/.test(location.pathname)) {
    bootPairedHome();
  } else {
    if (roomMatch) {
      try { localStorage.setItem("stp-room-token", roomMatch[1]); } catch (e) {}
    }
    loadMeta();
  }
})();
