package ui

import "encoding/json"

// PairedDevice is one saved phone (no token, no hash in the UI).
type PairedDevice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SetPairRename registers rename.
func (c *Control) SetPairRename(fn func(id, name string)) {
	c.mu.Lock()
	c.onRename = fn
	c.mu.Unlock()
}

// SetPairRemove registers remove.
func (c *Control) SetPairRemove(fn func(id string)) {
	c.mu.Lock()
	c.onRemove = fn
	c.mu.Unlock()
}

// PairRenameClicked is called from the WebView bind.
func (c *Control) PairRenameClicked(id, name string) {
	c.mu.Lock()
	fn := c.onRename
	c.mu.Unlock()
	if fn != nil {
		fn(id, name)
	}
}

// PairRemoveClicked is called from the WebView bind.
func (c *Control) PairRemoveClicked(id string) {
	c.mu.Lock()
	fn := c.onRemove
	c.mu.Unlock()
	if fn != nil {
		fn(id)
	}
}

// SetPaired redraws the paired-phone list (names only).
func (c *Control) SetPaired(devices []PairedDevice) {
	raw, err := json.Marshal(devices)
	if err != nil {
		return
	}
	js := `(function(){
	  var devices = ` + string(raw) + `;
	  var box = document.getElementById('paired');
	  var list = document.getElementById('paired-list');
	  if (!box || !list) return;
	  box.hidden = false;
	  list.innerHTML = '';
	  if (!devices.length) {
	    var empty = document.createElement('p');
	    empty.className = 'hint';
	    empty.textContent = 'No paired phones yet.';
	    list.appendChild(empty);
	    return;
	  }
	  devices.forEach(function(d){
	    var row = document.createElement('div');
	    row.className = 'dev-row';
	    var lab = document.createElement('span');
	    lab.textContent = d.name || 'Phone';
	    row.appendChild(lab);
	    var ren = document.createElement('button');
	    ren.type = 'button';
	    ren.textContent = 'Rename';
	    ren.onclick = function(){
	      var n = window.prompt('Name', d.name || 'Phone');
	      if (n && window.renamePair) window.renamePair(d.id, n);
	    };
	    row.appendChild(ren);
	    var rm = document.createElement('button');
	    rm.type = 'button';
	    rm.textContent = 'Remove';
	    rm.onclick = function(){ if (window.removePair) window.removePair(d.id); };
	    row.appendChild(rm);
	    list.appendChild(row);
	  });
	})()`
	c.onReady(func(eval func(string), _ func()) {
		if eval != nil {
			eval(js)
		}
	})
}
