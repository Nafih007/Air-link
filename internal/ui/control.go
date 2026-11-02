package ui

import (
	"encoding/json"
	"sync"
)

// Control lets other goroutines update or close the session window.
type Control struct {
	mu         sync.Mutex
	eval       func(js string)
	close      func()
	ready      chan struct{}
	closed     sync.Once
	onDecide   func(id string, accept bool)
	onOpenFold func()
	onKick     func(id string)
	onRename   func(id, name string)
	onRemove   func(id string)
}

// NewControl creates a control that becomes usable once the window attaches.
func NewControl() *Control {
	return &Control{ready: make(chan struct{})}
}

// Attach wires Eval/Terminate callbacks from the UI thread starter.
func (c *Control) Attach(eval func(js string), closeFn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eval = eval
	c.close = closeFn
	select {
	case <-c.ready:
	default:
		close(c.ready)
	}
}

func (c *Control) onReady(fn func(eval func(string), closeFn func())) {
	go func() {
		<-c.ready
		c.mu.Lock()
		eval := c.eval
		closeFn := c.close
		c.mu.Unlock()
		fn(eval, closeFn)
	}()
}

// ShowDone replaces the window content with a short Done state.
func (c *Control) ShowDone() {
	c.onReady(func(eval func(string), _ func()) {
		if eval == nil {
			return
		}
		eval(`(function(){
		  document.body.innerHTML = '<div style="font-family:Segoe UI,sans-serif;padding:2rem;text-align:center;background:#0f1419;color:#e7ecf3;min-height:100vh;box-sizing:border-box"><h1 style="color:#3dd68c">Done</h1><p style="color:#8b9bb4">Transfer finished. Closing…</p></div>';
		})()`)
	})
}

// SetStatus updates the live status line in the session window.
func (c *Control) SetStatus(text string) {
	raw, err := json.Marshal(text)
	if err != nil {
		return
	}
	js := `(function(){
	  var el = document.getElementById('status');
	  if (!el) return;
	  el.textContent = ` + string(raw) + `;
	  el.style.display = 'block';
	})()`
	c.onReady(func(eval func(string), _ func()) {
		if eval != nil {
			eval(js)
		}
	})
}

// SetProgress updates a simple progress label (bytes sent / total, or count-only).
func (c *Control) SetProgress(label string) {
	raw, err := json.Marshal(label)
	if err != nil {
		return
	}
	js := `(function(){
	  var el = document.getElementById('progress');
	  if (!el) return;
	  el.textContent = ` + string(raw) + `;
	  el.style.display = 'block';
	})()`
	c.onReady(func(eval func(string), _ func()) {
		if eval != nil {
			eval(js)
		}
	})
}

// ShowDiagnostics shows connection hints in the window.
func (c *Control) ShowDiagnostics(hints []string) {
	raw, err := json.Marshal(hints)
	if err != nil {
		return
	}
	js := `(function(){
	  var hints = ` + string(raw) + `;
	  var box = document.getElementById('diag');
	  if (!box) {
	    box = document.createElement('div');
	    box.id = 'diag';
	    box.style.cssText = 'margin-top:0.75rem;padding:0.75rem;border-radius:10px;background:#2a1518;color:#ffb4b4;font-size:0.85rem;line-height:1.4';
	    document.body.appendChild(box);
	  }
	  box.innerHTML = '<strong style="color:#ffd0d0">No phone connected yet</strong><ul style="margin:0.4rem 0 0;padding-left:1.1rem">' +
	    hints.map(function(h){ return '<li style="margin:0.25rem 0">'+h+'</li>'; }).join('') +
	    '</ul>';
	})()`
	c.onReady(func(eval func(string), _ func()) {
		if eval != nil {
			eval(js)
		}
	})
}

// SetUploadDecide registers the Accept/Decline handler for phone uploads.
func (c *Control) SetUploadDecide(fn func(id string, accept bool)) {
	c.mu.Lock()
	c.onDecide = fn
	c.mu.Unlock()
}

// UploadDecide is called from the WebView bind when the user clicks Accept/Decline.
func (c *Control) UploadDecide(id string, accept bool) {
	c.mu.Lock()
	fn := c.onDecide
	c.mu.Unlock()
	if fn != nil {
		fn(id, accept)
	}
}

// ShowUploadPrompt asks the user to Accept or Decline an incoming phone upload.
func (c *Control) ShowUploadPrompt(id, summary string) {
	idJSON, err1 := json.Marshal(id)
	sumJSON, err2 := json.Marshal(summary)
	if err1 != nil || err2 != nil {
		return
	}
	js := `(function(){
	  var id = ` + string(idJSON) + `;
	  var summary = ` + string(sumJSON) + `;
	  var box = document.getElementById('approve');
	  if (!box) return;
	  box.style.display = 'block';
	  box.innerHTML = '<strong>Phone wants to send</strong><p id="approve-msg"></p>' +
	    '<div class="approve-actions">' +
	    '<button type="button" class="approve-yes" id="approve-yes">Accept</button>' +
	    '<button type="button" class="approve-no" id="approve-no">Decline</button></div>';
	  document.getElementById('approve-msg').textContent = summary;
	  document.getElementById('approve-yes').onclick = function(){
	    if (window.uploadDecide) window.uploadDecide(id, true);
	    box.style.display = 'none';
	  };
	  document.getElementById('approve-no').onclick = function(){
	    if (window.uploadDecide) window.uploadDecide(id, false);
	    box.style.display = 'none';
	  };
	})()`
	c.onReady(func(eval func(string), _ func()) {
		if eval != nil {
			eval(js)
		}
	})
}

// HideUploadPrompt clears the approval panel.
func (c *Control) HideUploadPrompt() {
	js := `(function(){
	  var box = document.getElementById('approve');
	  if (box) box.style.display = 'none';
	})()`
	c.onReady(func(eval func(string), _ func()) {
		if eval != nil {
			eval(js)
		}
	})
}

// SetOpenFolder registers the handler for the Open folder button.
func (c *Control) SetOpenFolder(fn func()) {
	c.mu.Lock()
	c.onOpenFold = fn
	c.mu.Unlock()
}

// OpenFolderClicked is called from the WebView bind.
func (c *Control) OpenFolderClicked() {
	c.mu.Lock()
	fn := c.onOpenFold
	c.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// ShowClipboardToast shows a short notice without an Open folder button.
func (c *Control) ShowClipboardToast(message string) {
	msgJSON, err := json.Marshal(message)
	if err != nil {
		return
	}
	js := `(function(){
	  var message = ` + string(msgJSON) + `;
	  var box = document.getElementById('toast');
	  if (!box) return;
	  box.style.display = 'block';
	  box.innerHTML = '<strong id="toast-msg"></strong>';
	  document.getElementById('toast-msg').textContent = message;
	})()`
	c.onReady(func(eval func(string), _ func()) {
		if eval != nil {
			eval(js)
		}
	})
}

// ShowUploadToast shows a short success toast with an Open folder button.
func (c *Control) ShowUploadToast(message string) {
	msgJSON, err := json.Marshal(message)
	if err != nil {
		return
	}
	js := `(function(){
	  var message = ` + string(msgJSON) + `;
	  var box = document.getElementById('toast');
	  if (!box) return;
	  box.style.display = 'block';
	  box.innerHTML = '<strong id="toast-msg"></strong>' +
	    '<button type="button" class="toast-open" id="toast-open">Open folder</button>';
	  document.getElementById('toast-msg').textContent = message;
	  document.getElementById('toast-open').onclick = function(){
	    if (window.openUploadFolder) window.openUploadFolder();
	  };
	})()`
	c.onReady(func(eval func(string), _ func()) {
		if eval != nil {
			eval(js)
		}
	})
}

// SetKick registers the Kick-device handler.
func (c *Control) SetKick(fn func(id string)) {
	c.mu.Lock()
	c.onKick = fn
	c.mu.Unlock()
}

// KickClicked is called from the WebView bind.
func (c *Control) KickClicked(id string) {
	c.mu.Lock()
	fn := c.onKick
	c.mu.Unlock()
	if fn != nil {
		fn(id)
	}
}

// ConnectedDevice is one phone shown in the session window.
type ConnectedDevice struct {
	ID      string `json:"id"`
	IP      string `json:"ip"`
	Browser string `json:"browser"`
	Kicked  bool   `json:"kicked"`
}

// SetDevices redraws the connected-device list.
func (c *Control) SetDevices(devices []ConnectedDevice) {
	raw, err := json.Marshal(devices)
	if err != nil {
		return
	}
	js := `(function(){
	  var devices = ` + string(raw) + `;
	  var box = document.getElementById('devices');
	  var list = document.getElementById('device-list');
	  if (!box || !list) return;
	  box.hidden = devices.length === 0;
	  list.innerHTML = '';
	  devices.forEach(function(d){
	    var row = document.createElement('div');
	    row.className = 'dev-row';
	    var lab = document.createElement('span');
	    lab.textContent = d.ip + ' · ' + d.browser + (d.kicked ? ' (kicked)' : '');
	    row.appendChild(lab);
	    if (!d.kicked) {
	      var b = document.createElement('button');
	      b.type = 'button';
	      b.textContent = 'Kick';
	      b.onclick = function(){ if (window.kickDevice) window.kickDevice(d.id); };
	      row.appendChild(b);
	    }
	    list.appendChild(row);
	  });
	})()`
	c.onReady(func(eval func(string), _ func()) {
		if eval != nil {
			eval(js)
		}
	})
}

// Close terminates the UI loop (safe to call once from another goroutine).
func (c *Control) Close() {
	c.closed.Do(func() {
		c.onReady(func(_ func(string), closeFn func()) {
			if closeFn != nil {
				closeFn()
			}
		})
	})
}
