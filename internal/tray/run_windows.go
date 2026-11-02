//go:build windows

package tray

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/getlantern/systray"
	"sendtophone/internal/shell"
)

var (
	receiveMu  sync.Mutex
	receiveCmd *exec.Cmd
	selfExe    string
)

// Run shows the tray icon and blocks until Exit.
func Run(opt Options) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("tray: %w", err)
	}
	selfExe = exe
	systray.Run(func() { onReady(opt) }, func() { stopReceive() })
	return nil
}

func onReady(opt Options) {
	systray.SetIcon(appICO())
	systray.SetTitle("Air-Link")
	systray.SetTooltip("Air-Link")

	mRecv := systray.AddMenuItemCheckbox("Receive from phone", "Upload-only QR session", false)
	mShot := systray.AddMenuItem("Screenshot", "Capture the screen and show a QR")
	started, _ := shell.StartupEnabled()
	mBoot := systray.AddMenuItemCheckbox("Start with Windows", "Launch tray at logon", started)
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Quit the tray")

	var stopHK func()
	if opt.Hotkey {
		mods, vk, label, err := ParseCombo(opt.Combo)
		if err != nil {
			fmt.Fprintln(opt.Stderr, "airlink:", err)
			warnHotkey(err)
		} else {
			stopHK = listenScreenshotHotkey(mods, vk, label, func() {
				startChild("-screenshot")
			}, func(err error) {
				fmt.Fprintln(opt.Stderr, "airlink:", err)
				warnHotkey(err)
			})
		}
	}

	go func() {
		defer func() {
			if stopHK != nil {
				stopHK()
			}
			systray.Quit()
		}()
		for {
			select {
			case <-mRecv.ClickedCh:
				toggleReceive(mRecv)
			case <-mShot.ClickedCh:
				startChild("-screenshot")
			case <-mBoot.ClickedCh:
				toggleStartup(mBoot, opt.Stderr)
			case <-mQuit.ClickedCh:
				return
			}
		}
	}()
}

func toggleReceive(item *systray.MenuItem) {
	receiveMu.Lock()
	defer receiveMu.Unlock()
	if receiveCmd != nil && receiveCmd.Process != nil {
		_ = receiveCmd.Process.Kill()
		receiveCmd = nil
		item.Uncheck()
		return
	}
	cmd := exec.Command(selfExe, "-receive")
	if err := cmd.Start(); err != nil {
		item.Uncheck()
		return
	}
	receiveCmd = cmd
	item.Check()
	go func() {
		_ = cmd.Wait()
		receiveMu.Lock()
		if receiveCmd == cmd {
			receiveCmd = nil
		}
		receiveMu.Unlock()
		item.Uncheck()
	}()
}

func stopReceive() {
	receiveMu.Lock()
	defer receiveMu.Unlock()
	if receiveCmd != nil && receiveCmd.Process != nil {
		_ = receiveCmd.Process.Kill()
		receiveCmd = nil
	}
}

func startChild(args ...string) {
	cmd := exec.Command(selfExe, args...)
	_ = cmd.Start()
}

func toggleStartup(item *systray.MenuItem, stderr io.Writer) {
	on, _ := shell.StartupEnabled()
	var err error
	if on {
		err = shell.UninstallStartup()
		if err == nil {
			item.Uncheck()
		}
	} else {
		err = shell.InstallStartup(selfExe)
		if err == nil {
			item.Check()
		}
	}
	if err != nil && stderr != nil {
		fmt.Fprintln(stderr, "airlink:", err)
	}
}
