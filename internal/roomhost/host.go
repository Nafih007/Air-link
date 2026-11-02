package roomhost

import (
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	"sendtophone/internal/clipboard"
	"sendtophone/internal/mdns"
	"sendtophone/internal/netinfo"
	"sendtophone/internal/notify"
	"sendtophone/internal/pair"
	"sendtophone/internal/qr"
	"sendtophone/internal/server"
	"sendtophone/internal/ui"
)

// Item is one timeline row in the desktop window.
type Item struct {
	ID        string `json:"id"`
	Direction string `json:"direction"` // out | in
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Status    string `json:"status"` // ready | pending | done | failed | declined
	Detail    string `json:"detail"`
}

// Device is a connected phone shown in the desktop window.
type Device struct {
	ID      string `json:"id"`
	IP      string `json:"ip"`
	Browser string `json:"browser"`
	Kicked  bool   `json:"kicked"`
	Pending bool   `json:"pending"`
	Denied  bool   `json:"denied"`
}

// Snapshot is safe to send to the UI (includes the join URL for Copy only).
type Snapshot struct {
	QRDataURL   string   `json:"qrDataURL"`
	ListenAddr  string   `json:"listenAddr"`
	PageURL     string   `json:"pageURL"`
	PIN         string   `json:"pin"`
	Status      string   `json:"status"`
	UploadDir   string   `json:"uploadDir"`
	JoinLeftSec int      `json:"joinLeftSec"`
	AutoAllow   bool     `json:"autoAllow"`
	Devices     []Device `json:"devices"`
	Items       []Item   `json:"items"`
}

// Host is one connected-room HTTP session (M6) without a window.
type Host struct {
	mu       sync.Mutex
	opt      Options
	srv      *server.Server
	adv      *mdns.Advertiser
	qrPNG    []byte
	pageURL  string
	pin      string
	status   string
	started  time.Time
	stopIdle chan struct{}
	devices  []Device
	items    []Item
	seq      int
	onChange func()
}

// Options for a desktop room.
type Options struct {
	BindIP     string
	RequirePIN bool
	Idle       time.Duration
}

// StartRoom starts the M6 room server and builds a QR. Does not log the token.
func StartRoom(opt Options) (*Host, error) {
	ip := opt.BindIP
	if ip == "" {
		selected, _, err := netinfo.SelectLANIP()
		if err != nil {
			return nil, err
		}
		ip = selected.String()
	}
	tok, err := server.NewToken()
	if err != nil {
		return nil, err
	}
	var pin string
	if opt.RequirePIN {
		pin, err = server.NewPIN()
		if err != nil {
			return nil, err
		}
	}
	idle := opt.Idle
	if idle == 0 {
		idle = 30 * time.Minute
	}

	h := &Host{opt: opt, pin: pin, status: "Scan with your phone camera", started: time.Now(), stopIdle: make(chan struct{})}
	pairs, _ := pair.OpenDefault()

	srv, err := server.Start(server.Config{
		BindIP:     ip,
		Payload:    server.RoomPayload(),
		Token:      tok,
		PIN:        pin,
		Pairs:      pairs,
		AutoAccept: false,
		Hooks: server.Hooks{
			OnJoinConsumed: func() {
				h.rebuildQR()
				h.setStatus("Scan used — new QR ready")
			},
			OnPhoneHit: func() {
				h.setStatus("Connected")
			},
			OnTransferComplete: func() {
				h.markOutboundDone()
			},
			OnDevicesChanged: func(list []server.Device) {
				devs := make([]Device, 0, len(list))
				label := "Connected"
				for _, d := range list {
					devs = append(devs, Device{ID: d.ID, IP: d.IP, Browser: d.Browser, Kicked: d.Kicked, Pending: d.Pending, Denied: d.Denied})
					if d.Pending {
						label = "New device: " + d.Browser
					}
					if !d.Kicked && label == "Connected" {
						label = "Connected: " + d.Browser
					}
				}
				h.mu.Lock()
				h.devices = devs
				h.status = label
				h.mu.Unlock()
				h.notify()
			},
			OnUploadPending: func(id string, fileCount int, totalSize int64, hasText bool) {
				name := fmt.Sprintf("%d file(s)", fileCount)
				if hasText {
					name += " + text"
				}
				h.pushItem(Item{ID: id, Direction: "in", Name: name, Size: totalSize, Status: "pending", Detail: "Accept or decline"})
				h.setStatus("Phone sent files — Accept or Decline")
			},
			OnUploadSaved: func(fileCount int, totalSize int64) {
				h.finishInbound("done", fmt.Sprintf("Saved %d file(s)", fileCount))
				h.setStatus("Saved upload from phone")
				notify.Toast("Air-Link", fmt.Sprintf("Saved %d upload(s) from phone", fileCount))
			},
			OnClipboardText: func(text string) {
				_ = clipboard.WriteText(text)
				h.finishInbound("done", "Copied text to clipboard")
				h.setStatus("Copied text from phone")
			},
		},
	})
	if err != nil {
		return nil, err
	}
	h.srv = srv
	if err := h.rebuildQR(); err != nil {
		_ = srv.Close()
		return nil, err
	}
	if adv, err := mdns.Start(ip); err == nil {
		h.adv = adv
	}
	if idle > 0 {
		go h.watchIdle(idle)
	}
	return h, nil
}

func (h *Host) watchIdle(idle time.Duration) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-h.stopIdle:
			return
		case <-t.C:
			if h.srv == nil {
				return
			}
			exp := h.srv.JoinExpires()
			if !exp.IsZero() && time.Until(exp) <= 0 {
				_ = h.RefreshJoin()
			} else {
				h.notify()
			}
			last := h.srv.LastActivity()
			if last.IsZero() {
				last = h.started
			}
			if time.Since(last) >= idle {
				h.setStatus("Idle timeout — session ended")
				h.Close()
				return
			}
		}
	}
}

func (h *Host) setStatus(s string) {
	h.mu.Lock()
	h.status = s
	h.mu.Unlock()
	h.notify()
}

func (h *Host) notify() {
	h.mu.Lock()
	fn := h.onChange
	h.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// OnChange registers a UI refresh callback.
func (h *Host) OnChange(fn func()) {
	h.mu.Lock()
	h.onChange = fn
	h.mu.Unlock()
}

func (h *Host) pushItem(it Item) {
	h.mu.Lock()
	h.seq++
	if it.ID == "" {
		it.ID = fmt.Sprintf("i-%d", h.seq)
	}
	h.items = append(h.items, it)
	h.mu.Unlock()
	h.notify()
}

func (h *Host) markOutboundDone() {
	h.mu.Lock()
	for i := range h.items {
		if h.items[i].Direction == "out" && h.items[i].Status == "ready" {
			h.items[i].Status = "done"
			h.items[i].Detail = "Downloaded"
		}
	}
	h.mu.Unlock()
	h.notify()
}

func (h *Host) finishInbound(status, detail string) {
	h.mu.Lock()
	for i := range h.items {
		if h.items[i].Direction == "in" && h.items[i].Status == "pending" {
			h.items[i].Status = status
			h.items[i].Detail = detail
		}
	}
	h.mu.Unlock()
	h.notify()
}

// Snapshot for the Wails window. Caller must not log PageURL.
func (h *Host) Snapshot() Snapshot {
	h.mu.Lock()
	defer h.mu.Unlock()
	devs := append([]Device(nil), h.devices...)
	items := append([]Item(nil), h.items...)
	listen := ""
	up := ""
	left := 0
	auto := false
	if h.srv != nil {
		listen = h.srv.Addr()
		up = h.srv.UploadFolder()
		auto = h.srv.AutoAllow()
		sec := int(time.Until(h.srv.JoinExpires()).Seconds())
		if sec > 0 {
			left = sec
		}
	}
	return Snapshot{
		QRDataURL:   "data:image/png;base64," + base64.StdEncoding.EncodeToString(h.qrPNG),
		ListenAddr:  listen,
		PageURL:     h.pageURL,
		PIN:         h.pin,
		Status:      h.status,
		UploadDir:   up,
		JoinLeftSec: left,
		AutoAllow:   auto,
		Devices:     devs,
		Items:       items,
	}
}

// Enqueue adds PC files for the phone to download (existing session).
func (h *Host) Enqueue(paths []string) error {
	if h.srv == nil {
		return fmt.Errorf("session ended")
	}
	added, err := h.srv.EnqueuePaths(paths)
	if err != nil {
		return err
	}
	for _, f := range added {
		h.pushItem(Item{Direction: "out", Name: f.Name, Size: f.Size, Status: "ready", Detail: "On phone page"})
	}
	h.setStatus("Files added — phone can Save/Download")
	return nil
}

// AcceptUpload saves a pending phone batch.
func (h *Host) AcceptUpload(id string) error {
	if h.srv == nil {
		return fmt.Errorf("session ended")
	}
	_, err := h.srv.AcceptPending(id)
	return err
}

// DeclineUpload rejects a pending phone batch.
func (h *Host) DeclineUpload(id string) error {
	if h.srv == nil {
		return fmt.Errorf("session ended")
	}
	if err := h.srv.DeclinePending(id); err != nil {
		return err
	}
	h.mu.Lock()
	for i := range h.items {
		if h.items[i].ID == id {
			h.items[i].Status = "declined"
			h.items[i].Detail = "Declined"
		}
	}
	h.mu.Unlock()
	h.notify()
	return nil
}

// OpenReceivedFolder opens Downloads/Air-Link.
func (h *Host) OpenReceivedFolder() error {
	if h.srv == nil {
		return fmt.Errorf("session ended")
	}
	dir := h.srv.UploadFolder()
	_ = os.MkdirAll(dir, 0o755)
	return ui.OpenFolder(dir)
}

// Kick revokes a connected device (M4.2).
func (h *Host) Kick(id string) {
	if h.srv != nil {
		h.srv.KickDevice(id)
	}
}

// Allow accepts a pending phone.
func (h *Host) Allow(id string) {
	if h.srv != nil {
		h.srv.AllowDevice(id)
	}
}

// Deny rejects a pending phone.
func (h *Host) Deny(id string) {
	if h.srv != nil {
		h.srv.DenyDevice(id)
	}
}

// SetAutoAllow toggles skipping Allow on new scans.
func (h *Host) SetAutoAllow(on bool) {
	if h.srv != nil {
		h.srv.SetAutoAllow(on)
	}
}

// RefreshJoin invalidates the unused QR only.
func (h *Host) RefreshJoin() error {
	if h.srv == nil {
		return fmt.Errorf("session ended")
	}
	if err := h.srv.RotateJoin(); err != nil {
		return err
	}
	return h.rebuildQR()
}

func (h *Host) rebuildQR() error {
	if h.srv == nil {
		return fmt.Errorf("session ended")
	}
	u := h.srv.JoinURL()
	png, err := qr.PNGBytes(u, 512)
	if err != nil {
		return err
	}
	h.mu.Lock()
	h.pageURL = u
	h.qrPNG = png
	h.mu.Unlock()
	h.notify()
	return nil
}

// End stops HTTP so the QR and token 404.
func (h *Host) End() {
	h.setStatus("Session ended — QR no longer works")
	h.Close()
}

// Close stops HTTP and mDNS.
func (h *Host) Close() {
	h.mu.Lock()
	ch := h.stopIdle
	h.stopIdle = nil
	h.mu.Unlock()
	if ch != nil {
		close(ch)
	}
	if h.adv != nil {
		_ = h.adv.Close()
		h.adv = nil
	}
	if h.srv != nil {
		_ = h.srv.Close()
		h.srv = nil
	}
	h.notify()
}

// Alive reports whether the HTTP session is running.
func (h *Host) Alive() bool {
	return h != nil && h.srv != nil
}
