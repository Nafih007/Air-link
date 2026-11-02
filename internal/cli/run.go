package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sendtophone/internal/capture"
	"sendtophone/internal/clipboard"
	"sendtophone/internal/content"
	"sendtophone/internal/netinfo"
	"sendtophone/internal/server"
	"sendtophone/internal/shell"
	"sendtophone/internal/tray"
	"sendtophone/internal/wifi"
)

// Version is shown in CLI output (cmd/sendtophone may overwrite).
var Version = "dev"

// WantsCLI is true when args include flags (not only file paths).
func WantsCLI(args []string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			return true
		}
	}
	return false
}

// FileArgs returns non-flag arguments (paths).
func FileArgs(args []string) []string {
	var out []string
	for _, a := range args {
		if a == "--" {
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		out = append(out, a)
	}
	return out
}

// Run is the CLI entry (flags, receive, tray, one-shot send).
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("airlink", flag.ContinueOnError)
	fs.SetOutput(stderr)

	showVersion := fs.Bool("version", false, "print version and exit")
	fs.BoolVar(showVersion, "v", false, "print version and exit")
	bindIP := fs.String("bind", "", "LAN IPv4 to bind (default: auto-detect; never 0.0.0.0)")
	listIfaces := fs.Bool("list-ifaces", false, "list IPv4 interfaces and exit")
	idle := fs.Duration("idle", 10*time.Minute, "auto-shutdown after this much inactivity")
	diagnoseAfter := fs.Duration("diagnose-after", 20*time.Second, "show connection hints if no phone connects")
	keepAlive := fs.Bool("keep-alive", false, "keep serving after each download until idle/close")
	noWindow := fs.Bool("no-window", false, "skip desktop window; print QR in terminal only")
	installShell := fs.Bool("install-shell", false, "register right-click menu (same as -install-menu)")
	uninstallShell := fs.Bool("uninstall-shell", false, "remove right-click menu entries (same as -uninstall-menu)")
	installMenu := fs.Bool("install-menu", false, "register right-click Send with Air-Link + SendTo shortcut")
	uninstallMenu := fs.Bool("uninstall-menu", false, "remove right-click Send with Air-Link and SendTo shortcut")
	sendClipboard := fs.Bool("clipboard", false, "send clipboard text (short → direct QR; long → page)")
	sendClipImage := fs.Bool("clipboard-image", false, "send clipboard image as a PNG file")
	textFlag := fs.String("text", "", "send this text (same rules as -clipboard)")
	screenshot := fs.Bool("screenshot", false, "capture fullscreen screenshot and send as PNG")
	shareWifi := fs.Bool("wifi", false, "share current Wi-Fi as WIFI: QR (asks before reading password)")
	autoAccept := fs.Bool("auto-accept", false, "save phone uploads without PC approval (M3.3)")
	uploadDir := fs.String("upload-dir", "", "folder for phone uploads (default: Downloads\\Air-Link)")
	receive := fs.Bool("receive", false, "upload-only session: QR for phone → PC (no files to download)")
	convertJPG := fs.Bool("convert-jpg", false, "convert uploaded photos (HEIC/PNG/…) to JPEG; off = keep original")
	requirePIN := fs.Bool("pin", false, "require a 4-digit PIN on the phone (shown on the PC)")
	runTray := fs.Bool("tray", false, "keep a tray icon (Receive on/off, Screenshot, Exit)")
	hotkey := fs.Bool("hotkey", false, "enable screenshot hotkey (implies -tray; off by default)")
	hotkeyCombo := fs.String("hotkey-combo", tray.DefaultHotkey, "screenshot combo when -hotkey is set")
	port := fs.Int("port", 0, "TCP port (0 = random; -receive defaults to 17612)")
	maxUpload := fs.Int64("max-upload", 0, "max bytes per uploaded file (0 = 512MiB)")
	installStartup := fs.Bool("install-startup", false, "start tray with Windows (HKCU Run)")
	uninstallStartup := fs.Bool("uninstall-startup", false, "remove start-with-Windows")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: airlink [flags] [-- <path> [path...]]\n\n")
		fmt.Fprintf(stderr, "No args: open a room QR and wait for a phone. Or pass files, -receive, -tray, …\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	if *showVersion {
		fmt.Fprintf(stdout, "Air-Link v%s\n", Version)
		return 0
	}
	if *listIfaces {
		return printIfaces(stdout, stderr)
	}
	if *installMenu || *installShell {
		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(stderr, "airlink: %v\n", err)
			return 1
		}
		if !shell.ShouldAutoRegister(exe) {
			fmt.Fprintf(stderr, "airlink: refusing to register a temporary go run binary; use the Wails exe (build\\bin\\airlink.exe).\n")
			return 1
		}
		if err := shell.InstallContextMenu(exe); err != nil {
			fmt.Fprintf(stderr, "airlink: install-menu: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Installed \"Send with Air-Link\" (Shift+right-click / Show more options) and the SendTo shortcut.\n")
		fmt.Fprintf(stdout, "Windows 11 short menu is not registered (needs a signed package).\n")
		return 0
	}
	if *uninstallMenu || *uninstallShell {
		if err := shell.UninstallContextMenu(); err != nil {
			fmt.Fprintf(stderr, "airlink: uninstall-menu: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Removed \"Send with Air-Link\" menu entries and the SendTo shortcut.\n")
		return 0
	}
	if *installStartup {
		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(stderr, "airlink: %v\n", err)
			return 1
		}
		if err := shell.InstallStartup(exe); err != nil {
			fmt.Fprintf(stderr, "airlink: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Will start the tray at Windows logon.\n")
		return 0
	}
	if *uninstallStartup {
		if err := shell.UninstallStartup(); err != nil {
			fmt.Fprintf(stderr, "airlink: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Removed start-with-Windows.\n")
		return 0
	}

	opt := sessionOpts{
		bindIP:        *bindIP,
		idle:          *idle,
		diagnoseAfter: *diagnoseAfter,
		keepAlive:     *keepAlive,
		noWindow:      *noWindow,
		autoAccept:    *autoAccept,
		uploadDir:     *uploadDir,
		convertJPG:    *convertJPG,
		requirePIN:    *requirePIN,
		maxUpload:     *maxUpload,
		port:          *port,
		stdout:        stdout,
		stderr:        stderr,
	}

	if *receive {
		opt.keepAlive = true
		idleSet := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "idle" {
				idleSet = true
			}
		})
		if !idleSet {
			opt.idle = 0
		}
		if opt.port == 0 {
			opt.port = server.DefaultReceivePort
		}
		return runFileOrTextSession(server.ReceivePayload(), opt)
	}
	if *screenshot {
		return runScreenshot(opt)
	}
	if *shareWifi {
		return runShareWifi(opt)
	}
	if *sendClipImage {
		return runClipboardImage(opt)
	}
	if *sendClipboard || *textFlag != "" {
		text := *textFlag
		var err error
		if *sendClipboard {
			text, err = clipboard.ReadText()
			if err != nil {
				fmt.Fprintf(stderr, "airlink: %v\n", err)
				return 1
			}
		}
		text, err = content.NormalizeClipboardText(text)
		if err != nil {
			fmt.Fprintf(stderr, "airlink: %v\n", err)
			return 1
		}
		return runClipboardText(text, opt)
	}

	if *runTray || *hotkey {
		if err := tray.Run(tray.Options{Hotkey: *hotkey, Combo: *hotkeyCombo, Stdout: stdout, Stderr: stderr}); err != nil {
			fmt.Fprintf(stderr, "airlink: %v\n", err)
			return 1
		}
		return 0
	}

	paths := fs.Args()
	if len(paths) == 0 {
		if shell.LiveInstanceExists() {
			_ = shell.OfferPaths(nil)
			shell.FocusMainWindow()
			return 0
		}
		idleSet := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "idle" {
				idleSet = true
			}
		})
		if !idleSet {
			opt.idle = 30 * time.Minute
		}
		opt.keepAlive = true
		return runFileOrTextSession(server.RoomPayload(), opt)
	}
	if shell.LiveInstanceExists() {
		_ = shell.OfferPaths(paths)
		shell.FocusMainWindow()
		return 0
	}
	paths, leader, err := shell.GatherPaths(paths)
	if err != nil {
		fmt.Fprintf(stderr, "airlink: %v\n", err)
		return 1
	}
	if !leader {
		return 0
	}
	payload, err := server.ResolvePaths(paths)
	if err != nil {
		fmt.Fprintf(stderr, "airlink: %v\n", err)
		return 1
	}
	return runFileOrTextSession(payload, opt)
}

func runClipboardText(text string, opt sessionOpts) int {
	if content.FitsDirectQR(text) {
		label := "Clipboard text"
		if content.LooksLikeURL(text) {
			label = "Clipboard URL (scan to open)"
		}
		return runDirectQR(text, label, opt)
	}
	return runFileOrTextSession(server.TextPayload(text), opt)
}

func runClipboardImage(opt sessionOpts) int {
	png, err := clipboard.ReadImagePNG()
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	dir, err := os.MkdirTemp("", "sendtophone-clip-*")
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "clipboard.png")
	if err := os.WriteFile(path, png, 0o600); err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	payload, err := server.ResolvePaths([]string{path})
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	payload.DisplayName = "Clipboard image"
	return runFileOrTextSession(payload, opt)
}

func runScreenshot(opt sessionOpts) int {
	path, cleanup, err := capture.FullscreenPNG()
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	defer cleanup()
	payload, err := server.ResolvePaths([]string{path})
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	payload.DisplayName = "Screenshot"
	return runFileOrTextSession(payload, opt)
}

func runShareWifi(opt sessionOpts) int {
	ssid, err := wifi.CurrentProfile()
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	fmt.Fprintf(opt.stdout, "Share Wi-Fi password for %q with a QR code?\n", ssid)
	fmt.Fprintf(opt.stdout, "The password is read only if you confirm, and is not stored.\n")
	fmt.Fprintf(opt.stdout, "Type y to confirm: ")
	var ans string
	if _, err := fmt.Fscanln(os.Stdin, &ans); err != nil || !isYes(ans) {
		fmt.Fprintf(opt.stdout, "Cancelled.\n")
		return 0
	}
	pass, err := wifi.ReadPassword(ssid)
	if err != nil {
		fmt.Fprintf(opt.stderr, "airlink: %v\n", err)
		return 1
	}
	payload := wifi.QRPayload(wifi.Credentials{SSID: ssid, Password: pass})
	return runDirectQR(payload, "Wi-Fi: "+ssid, opt)
}

func isYes(s string) bool {
	switch s {
	case "y", "Y", "yes", "YES", "Yes":
		return true
	default:
		return false
	}
}

func printIfaces(stdout, stderr io.Writer) int {
	list, err := netinfo.ListIPv4()
	if err != nil {
		fmt.Fprintf(stderr, "airlink: %v\n", err)
		return 1
	}
	if len(list) == 0 {
		fmt.Fprintf(stdout, "No IPv4 interfaces found.\n")
		return 0
	}
	fmt.Fprintf(stdout, "IPv4 interfaces (* = skipped for auto-bind):\n")
	for _, a := range list {
		mark := " "
		if netinfo.SkipAdapter(a.Name) {
			mark = "*"
		}
		fmt.Fprintf(stdout, "%s %-24s %s\n", mark, a.Name, a.IP)
	}
	return 0
}
