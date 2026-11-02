package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"airlink/internal/clipboard"
	"airlink/internal/mdns"
	"airlink/internal/netinfo"
	"airlink/internal/notify"
	"airlink/internal/pair"
	"airlink/internal/qr"
	"airlink/internal/server"
	"airlink/internal/ui"
)

type sessionOpts struct {
	bindIP         string
	idle           time.Duration
	diagnoseAfter  time.Duration
	keepAlive      bool
	noWindow       bool
	autoAccept     bool
	uploadDir      string
	convertJPG     bool
	requirePIN     bool
	maxUpload      int64
	port           int
	stdout, stderr io.Writer
}

func resolveBind(bindIP string) (ip, ifaceName string, err error) {
	if bindIP != "" {
		return bindIP, "", nil
	}
	selected, name, err := netinfo.SelectLANIP()
	if err != nil {
		return "", "", err
	}
	return selected.String(), name, nil
}

func runFileOrTextSession(payload server.Payload, opt sessionOpts) int {
	ip, ifaceName, err := resolveBind(opt.bindIP)
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		fmt.Fprintf(opt.stderr, "Hint: run with -list-ifaces, then pass -bind <ip>\n")
		return 1
	}
	tok, err := server.NewToken()
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	var pin string
	if opt.requirePIN {
		pin, err = server.NewPIN()
		if err != nil {
			fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
			return 1
		}
	}

	ctrl := ui.NewControl()
	stopCh := make(chan struct{})
	var stopOnce sync.Once
	requestStop := func() {
		stopOnce.Do(func() { close(stopCh) })
	}

	pairs, perr := pair.OpenDefault()
	if perr != nil {
		fmt.Fprintf(opt.stderr, "airlink: pair store: %v\n", perr)
		pairs = nil
	}

	var srv *server.Server
	notifyUploadSaved := func(fileCount int, totalSize int64) {
		folder := ""
		if srv != nil {
			folder = srv.UploadFolder()
		}
		msg := fmt.Sprintf("Saved %d upload(s) from phone (%s)", fileCount, ui.FormatSize(totalSize))
		fmt.Fprintf(opt.stdout, "%s\n", msg)
		if folder != "" {
			fmt.Fprintf(opt.stdout, "Folder: %s\n", folder)
		}
		ctrl.SetStatus(msg)
		ctrl.HideUploadPrompt()
		ctrl.ShowUploadToast(msg)
		notify.Toast("Air-Link", msg)
	}

	refreshPaired := func() {
		if pairs == nil {
			return
		}
		list := pairs.List()
		devs := make([]ui.PairedDevice, 0, len(list))
		for _, r := range list {
			devs = append(devs, ui.PairedDevice{ID: r.ID, Name: r.Name})
		}
		ctrl.SetPaired(devs)
	}
	ctrl.SetPairRename(func(id, name string) {
		if pairs == nil {
			return
		}
		if err := pairs.Rename(id, name); err != nil {
			fmt.Fprintf(opt.stderr, "airlink: rename: %v\n", err)
			return
		}
		refreshPaired()
	})
	ctrl.SetPairRemove(func(id string) {
		if pairs == nil {
			return
		}
		if err := pairs.Remove(id); err != nil {
			fmt.Fprintf(opt.stderr, "airlink: remove: %v\n", err)
			return
		}
		refreshPaired()
		ctrl.SetStatus("Removed paired phone")
	})

	ctrl.SetKick(func(id string) {
		if srv != nil {
			srv.KickDevice(id)
			ctrl.SetStatus("Kicked device")
		}
	})

	ctrl.SetOpenFolder(func() {
		if srv == nil {
			return
		}
		folder := srv.UploadFolder()
		if folder == "" {
			return
		}
		_ = os.MkdirAll(folder, 0o755)
		if err := ui.OpenFolder(folder); err != nil {
			fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		}
	})

	ctrl.SetUploadDecide(func(id string, accept bool) {
		if srv == nil {
			return
		}
		if accept {
			saved, err := srv.AcceptPending(id)
			if err != nil {
				fmt.Fprintf(opt.stderr, "airlink: accept upload: %v\n", err)
				ctrl.SetStatus("Accept failed")
				return
			}
			fmt.Fprintf(opt.stdout, "Accepted upload → %v\n", saved)
			return
		}
		if err := srv.DeclinePending(id); err != nil {
			fmt.Fprintf(opt.stderr, "airlink: decline upload: %v\n", err)
			ctrl.SetStatus("Decline failed")
			return
		}
		fmt.Fprintf(opt.stdout, "Declined upload %s\n", id)
		ctrl.SetStatus("Declined phone upload")
	})

	srv, err = server.Start(server.Config{
		BindIP:       ip,
		Payload:      payload,
		Token:        tok,
		UploadDir:    opt.uploadDir,
		AutoAccept:   opt.autoAccept,
		ConvertToJPG: opt.convertJPG,
		PIN:          pin,
		Pairs:        pairs,
		MaxUpload:    opt.maxUpload,
		Port:         opt.port,
		Hooks: server.Hooks{
			OnPhoneHit: func() { ctrl.SetStatus("Phone connected") },
			OnProgress: func(written, total int64) {
				if total > 0 {
					ctrl.SetProgress(fmt.Sprintf("Sending %s / %s", ui.FormatSize(written), ui.FormatSize(total)))
					return
				}
				if written > 0 {
					ctrl.SetProgress(fmt.Sprintf("Sending %s…", ui.FormatSize(written)))
				}
			},
			OnUploadProgress: func(got, total int64) {
				if total > 0 {
					ctrl.SetProgress(fmt.Sprintf("Receiving %s / %s", ui.FormatSize(got), ui.FormatSize(total)))
					return
				}
				ctrl.SetProgress(fmt.Sprintf("Receiving %s…", ui.FormatSize(got)))
			},
			OnTransferComplete: func() {
				n := srv.TransferCount()
				if opt.keepAlive {
					msg := fmt.Sprintf("Sent ×%d — keep-alive on, waiting for more", n)
					fmt.Fprintf(opt.stdout, "%s\n", msg)
					ctrl.SetStatus(msg)
					ctrl.SetProgress("")
					return
				}
				fmt.Fprintf(opt.stdout, "Done\n")
				ctrl.ShowDone()
				go func() {
					time.Sleep(2 * time.Second)
					requestStop()
					ctrl.Close()
				}()
			},
			OnUploadPending: func(id string, fileCount int, totalSize int64, hasText bool) {
				summary := formatUploadSummary(fileCount, totalSize, hasText)
				fmt.Fprintf(opt.stdout, "Incoming upload %s: %s\n", id, summary)
				if opt.noWindow {
					go promptUploadCLI(id, summary, opt, srv, ctrl)
					return
				}
				ctrl.ShowUploadPrompt(id, summary)
				ctrl.SetStatus("Waiting for your Accept / Decline")
			},
			OnUploadSaved: notifyUploadSaved,
			OnClipboardText: func(text string) {
				if err := clipboard.WriteText(text); err != nil {
					fmt.Fprintf(opt.stderr, "airlink: clipboard: %v\n", err)
					ctrl.SetStatus("Could not copy text to clipboard")
					return
				}
				msg := "Copied text from phone to clipboard"
				fmt.Fprintf(opt.stdout, "%s\n", msg)
				ctrl.HideUploadPrompt()
				ctrl.SetStatus(msg)
				ctrl.ShowClipboardToast(msg)
			},
			OnDevicesChanged: func(list []server.Device) {
				devs := make([]ui.ConnectedDevice, 0, len(list))
				for _, d := range list {
					devs = append(devs, ui.ConnectedDevice{
						ID: d.ID, IP: d.IP, Browser: d.Browser, Kicked: d.Kicked,
					})
				}
				ctrl.SetDevices(devs)
			},
			OnPairsChanged: func(_ []pair.Record) {
				refreshPaired()
			},
		},
	})
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	defer srv.Close()
	refreshPaired()

	adv, merr := mdns.Start(ip)
	if merr != nil {
		fmt.Fprintf(opt.stderr, "airlink: mDNS (optional): %v\n", merr)
	} else {
		defer adv.Close()
	}

	sessionURL := srv.SessionURL()
	qrPNG, err := qr.PNGBytes(sessionURL, 512)
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	qrText, err := qr.Terminal(sessionURL)
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	qrPath := filepath.Join(os.TempDir(), "airlink-qr.png")
	_ = qr.WritePNG(sessionURL, qrPath, 512)
	defer func() { _ = os.Remove(qrPath) }()

	fmt.Fprintf(opt.stdout, "Air-Link v%s\n", Version)
	if payload.Room {
		fmt.Fprintf(opt.stdout, "Mode: connected room (waiting for phone)\n")
	} else if payload.ReceiveOnly {
		fmt.Fprintf(opt.stdout, "Mode: receive from phone (upload only)\n")
	} else {
		fmt.Fprintf(opt.stdout, "Sending: %s (%s)\n", payload.DisplayName, ui.FormatSize(payload.Size))
	}
	if ifaceName != "" {
		fmt.Fprintf(opt.stdout, "Interface: %s\n", ifaceName)
	}
	fmt.Fprintf(opt.stdout, "Listening on %s\n", srv.Addr())
	_, listenPort, _ := net.SplitHostPort(srv.Addr())
	if adv != nil {
		fmt.Fprintf(opt.stdout, "mDNS: http://%s:%s/ (IP URL is the fallback if DHCP changes or .local fails)\n", adv.HostnameLocal(), listenPort)
	}
	if payload.ReceiveOnly || payload.Room {
		fmt.Fprintf(opt.stdout, "Home Screen / rejoin: %s\n", srv.HomeURL())
	}
	if opt.idle <= 0 {
		fmt.Fprintf(opt.stdout, "Idle timeout: off (close the window to stop)\n")
	} else if opt.keepAlive {
		fmt.Fprintf(opt.stdout, "Keep-alive: on (close window or wait for idle to stop)\n")
	}
	if opt.autoAccept {
		fmt.Fprintf(opt.stdout, "Uploads: auto-accept on\n")
	} else {
		fmt.Fprintf(opt.stdout, "Uploads: Accept/Decline required (or pass -auto-accept)\n")
	}
	if opt.convertJPG {
		fmt.Fprintf(opt.stdout, "Uploads: convert photos to JPEG (HEIC kept if convert fails)\n")
	}
	if pin != "" {
		fmt.Fprintf(opt.stdout, "PIN required on phone: %s\n", pin)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchSession(ctx, srv, ctrl, opt.stdout, opt.idle, opt.diagnoseAfter, requestStop)

	if opt.noWindow {
		fmt.Fprintf(opt.stdout, "\n%s\n", qrText)
		fmt.Fprintf(opt.stdout, "QR image: %s\n", qrPath)
		fmt.Fprintf(opt.stdout, "Or open in Safari on the phone:\n%s\n", sessionURL)
		fmt.Fprintf(opt.stdout, "Press Ctrl+C to stop.\n")
		ctrl.Attach(func(string) {}, requestStop)
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		select {
		case <-stopCh:
		case <-sig:
		}
		fmt.Fprintf(opt.stdout, "Stopping.\n")
		return 0
	}

	fmt.Fprintf(opt.stdout, "Opening session window (close it to stop).\n")
	onExp := func() { requestStop() }
	if opt.idle <= 0 {
		onExp = nil
	}
	err = ui.RunSessionWindow(ui.Session{
		FileName:   payload.DisplayName,
		FileSize:   payload.Size,
		ListenAddr: srv.Addr(),
		QRPNG:      qrPNG,
		IdleFor:    opt.idle,
		PIN:        pin,
		OnExpired:  onExp,
	}, ctrl)
	requestStop()
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		fmt.Fprintf(opt.stderr, "Fallback: use -no-window\n")
		return 1
	}
	fmt.Fprintf(opt.stdout, "Stopping.\n")
	return 0
}

func watchSession(
	ctx context.Context,
	srv *server.Server,
	ctrl *ui.Control,
	stdout io.Writer,
	idle, diagnoseAfter time.Duration,
	requestStop func(),
) {
	started := time.Now()
	var diagnosed sync.Once
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if diagnoseAfter > 0 && time.Since(started) >= diagnoseAfter {
				diagnosed.Do(func() {
					if srv.PhoneHit() {
						return
					}
					hints := server.ConnectionHints()
					fmt.Fprintf(stdout, "\nNo phone connected after %s:\n", diagnoseAfter)
					for _, h := range hints {
						fmt.Fprintf(stdout, "  • %s\n", h)
					}
					ctrl.ShowDiagnostics(hints)
				})
			}
			if idle <= 0 {
				continue
			}
			last := srv.LastActivity()
			if last.IsZero() {
				last = started
			}
			if time.Since(last) >= idle {
				fmt.Fprintf(stdout, "Idle timeout (%s) — shutting down.\n", idle)
				requestStop()
				ctrl.Close()
				return
			}
		}
	}
}

func formatUploadSummary(fileCount int, totalSize int64, hasText bool) string {
	parts := fmt.Sprintf("%d file(s), %s", fileCount, ui.FormatSize(totalSize))
	if hasText {
		parts += " + text"
	}
	return parts
}

func promptUploadCLI(id, summary string, opt sessionOpts, srv *server.Server, ctrl *ui.Control) {
	fmt.Fprintf(opt.stdout, "Phone wants to send %s. Accept? [y/N]: ", summary)
	var ans string
	if _, err := fmt.Fscanln(os.Stdin, &ans); err != nil || !isYes(ans) {
		_ = srv.DeclinePending(id)
		fmt.Fprintf(opt.stdout, "Declined.\n")
		ctrl.SetStatus("Declined phone upload")
		return
	}
	saved, err := srv.AcceptPending(id)
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: accept upload: %v\n", err)
		return
	}
	fmt.Fprintf(opt.stdout, "Accepted → %v\n", saved)
}

// runDirectQR shows a QR that encodes content itself (no HTTP server).
func runDirectQR(content, label string, opt sessionOpts) int {
	qrPNG, err := qr.PNGBytes(content, 512)
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	qrText, err := qr.Terminal(content)
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	qrPath := filepath.Join(os.TempDir(), "airlink-qr.png")
	_ = qr.WritePNG(content, qrPath, 512)
	defer func() { _ = os.Remove(qrPath) }()

	fmt.Fprintf(opt.stdout, "Air-Link v%s\n", Version)
	fmt.Fprintf(opt.stdout, "Direct QR (no server): %s\n", label)
	fmt.Fprintf(opt.stdout, "Content length: %d characters\n", len([]rune(content)))

	if opt.noWindow {
		fmt.Fprintf(opt.stdout, "\n%s\n", qrText)
		fmt.Fprintf(opt.stdout, "QR image: %s\n", qrPath)
		fmt.Fprintf(opt.stdout, "Scan with the phone camera — no download server is running.\n")
		fmt.Fprintf(opt.stdout, "Press Ctrl+C to exit.\n")
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		return 0
	}

	ctrl := ui.NewControl()
	fmt.Fprintf(opt.stdout, "Opening QR window (close when done).\n")
	err = ui.RunSessionWindow(ui.Session{
		FileName:   label,
		FileSize:   int64(len(content)),
		ListenAddr: "direct QR — no server",
		QRPNG:      qrPNG,
		IdleFor:    opt.idle,
	}, ctrl)
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	return 0
}
