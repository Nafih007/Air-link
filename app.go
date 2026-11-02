package main

import (
	"context"
	"fmt"
	"time"

	"sendtophone/internal/applog"
	"sendtophone/internal/clipboard"
	"sendtophone/internal/roomhost"
	"sendtophone/internal/shell"
	"sendtophone/internal/tray"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails front-end for the existing room server.
type App struct {
	ctx       context.Context
	host      *roomhost.Host
	forceQuit bool
	pending   []string
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = applog.Open()
	applog.Info("window start")
	if err := shell.EnsureContextMenu(""); err != nil {
		applog.Info("context menu: %s", err.Error())
	}
	if err := a.startRoom(); err != nil {
		runtime.LogError(ctx, err.Error())
		applog.Info("room start failed: %s", err.Error())
		runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Air-Link",
			Message: err.Error() + "\n\nUse airlink.exe -list-ifaces then -bind <ip> if the LAN IP cannot be detected.",
		})
		return
	}
	runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
		_ = a.SendPaths(paths)
	})
	wait := time.Duration(0)
	if len(a.pending) > 0 {
		wait = shell.MultiLaunchWindow
	}
	batch := shell.CollectIncoming(a.pending, wait)
	a.pending = nil
	if len(batch) > 0 {
		_ = a.SendPaths(batch)
	}
	go a.watchHandoff(ctx)
}

func (a *App) startRoom() error {
	if a.host != nil {
		a.host.Close()
	}
	h, err := roomhost.StartRoom(roomhost.Options{})
	if err != nil {
		return err
	}
	a.host = h
	h.OnChange(func() {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "room:update", h.Snapshot())
		}
	})
	applog.Info("room listening on %s", h.Snapshot().ListenAddr)
	return nil
}

func (a *App) watchHandoff(ctx context.Context) {
	t := time.NewTicker(80 * time.Millisecond)
	defer t.Stop()
	var batch []string
	var flushAt time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			extra := shell.PollOfferedPaths()
			if len(extra) > 0 {
				if len(batch) == 0 {
					flushAt = time.Now().Add(shell.MultiLaunchWindow)
				}
				batch = append(batch, extra...)
			}
			if len(batch) == 0 || time.Now().Before(flushAt) {
				continue
			}
			runtime.WindowShow(a.ctx)
			runtime.WindowUnminimise(a.ctx)
			_ = a.SendPaths(batch)
			batch = nil
		}
	}
}

func (a *App) shutdown(ctx context.Context) {
	applog.Info("shutdown")
	if a.host != nil {
		a.host.Close()
		a.host = nil
	}
	shell.ReleaseLiveInstance()
	applog.Close()
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.forceQuit {
		return false
	}
	choice, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "Air-Link",
		Message:       "Keep receiving in the background?",
		Buttons:       []string{"Keep in tray", "Quit"},
		DefaultButton: "Keep in tray",
		CancelButton:  "Quit",
	})
	if err != nil {
		return false
	}
	if choice != "Keep in tray" {
		return false
	}
	tray.StartEmbedded(tray.Embedded{
		OnShow: func() {
			runtime.WindowShow(a.ctx)
			runtime.WindowUnminimise(a.ctx)
		},
		OnOpenFolder: func() { _ = a.OpenReceivedFolder() },
		OnQuit: func() {
			a.forceQuit = true
			runtime.Quit(a.ctx)
		},
	})
	runtime.WindowHide(ctx)
	applog.Info("hidden to tray")
	return true
}

// Snapshot returns QR, URL, PIN, status, devices and timeline.
func (a *App) Snapshot() roomhost.Snapshot {
	if a.host == nil {
		return roomhost.Snapshot{Status: "Starting…"}
	}
	return a.host.Snapshot()
}

// KickDevice revokes a phone (same as M4.2).
func (a *App) KickDevice(id string) {
	if a.host != nil {
		a.host.Kick(id)
	}
}

// AllowDevice accepts a pending join.
func (a *App) AllowDevice(id string) {
	if a.host != nil {
		a.host.Allow(id)
	}
}

// DenyDevice rejects a pending join.
func (a *App) DenyDevice(id string) {
	if a.host != nil {
		a.host.Deny(id)
	}
}

// SetAutoAllow skips Allow prompts when on.
func (a *App) SetAutoAllow(on bool) {
	if a.host != nil {
		a.host.SetAutoAllow(on)
	}
}

// CopyPageURL copies the session URL for pasting into Safari. Not logged.
func (a *App) CopyPageURL() string {
	if a.host == nil {
		return "not ready"
	}
	u := a.host.Snapshot().PageURL
	if err := clipboard.WriteText(u); err != nil {
		return fmt.Sprintf("%v", err)
	}
	return "copied"
}

// PickFiles opens a multi-file dialog and enqueues the selection.
func (a *App) PickFiles() string {
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Send files with Air-Link",
	})
	if err != nil {
		return err.Error()
	}
	return a.SendPaths(paths)
}

// SendPaths enqueues files onto the open room.
func (a *App) SendPaths(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if a.host == nil || !a.host.Alive() {
		return "session ended — End session / start again first"
	}
	if err := a.host.Enqueue(paths); err != nil {
		applog.Info("enqueue failed: %s", err.Error())
		return err.Error()
	}
	applog.Info("enqueued %d path(s)", len(paths))
	return ""
}

// AcceptUpload saves a pending phone batch.
func (a *App) AcceptUpload(id string) string {
	if a.host == nil {
		return "not ready"
	}
	if err := a.host.AcceptUpload(id); err != nil {
		return err.Error()
	}
	return ""
}

// DeclineUpload rejects a pending phone batch.
func (a *App) DeclineUpload(id string) string {
	if a.host == nil {
		return "not ready"
	}
	if err := a.host.DeclineUpload(id); err != nil {
		return err.Error()
	}
	return ""
}

// OpenReceivedFolder opens the upload directory.
func (a *App) OpenReceivedFolder() string {
	if a.host == nil {
		return "not ready"
	}
	if err := a.host.OpenReceivedFolder(); err != nil {
		return err.Error()
	}
	return ""
}

// EndSession stops HTTP so the QR 404s.
func (a *App) EndSession() {
	if a.host != nil {
		a.host.End()
		applog.Info("session ended")
	}
}

// ContextMenu returns right-click / SendTo install status for Settings.
func (a *App) ContextMenu() shell.MenuStatus {
	return shell.ContextMenuStatus()
}

// InstallMenu registers the classic verb and SendTo shortcut.
func (a *App) InstallMenu() string {
	if err := shell.InstallContextMenu(""); err != nil {
		return err.Error()
	}
	applog.Info("context menu installed")
	return ""
}

// RepairMenu re-registers the menu using this exe's current path.
func (a *App) RepairMenu() string {
	return a.InstallMenu()
}

// UninstallMenu removes the classic verb and SendTo shortcut.
func (a *App) UninstallMenu() string {
	if err := shell.UninstallContextMenu(); err != nil {
		return err.Error()
	}
	applog.Info("context menu removed")
	return ""
}

// RefreshQR invalidates the unused join code only.
func (a *App) RefreshQR() string {
	if a.host == nil {
		return "not ready"
	}
	if err := a.host.RefreshJoin(); err != nil {
		return err.Error()
	}
	applog.Info("join QR refreshed")
	return ""
}
