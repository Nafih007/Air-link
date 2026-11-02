//go:build windows

package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func verbTargets(name string) []string {
	return []string{
		`Software\Classes\*\shell\` + name,
		`Software\Classes\Directory\shell\` + name,
	}
}

func currentExe(commandExe string) (string, error) {
	exe := commandExe
	var err error
	if exe == "" {
		exe, err = os.Executable()
		if err != nil {
			return "", fmt.Errorf("executable: %w", err)
		}
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return "", fmt.Errorf("resolve exe: %w", err)
	}
	return exe, nil
}

// InstallContextMenu writes HKCU classic verbs, a SendTo shortcut, and notifies Explorer.
func InstallContextMenu(commandExe string) error {
	exe, err := currentExe(commandExe)
	if err != nil {
		return err
	}
	_ = deleteVerbKeys(oldVerbName)
	_ = deleteVerbKeys(legacyVerb)
	for _, keyPath := range verbTargets(verbName) {
		if err := writeVerb(keyPath, exe); err != nil {
			return err
		}
	}
	if err := installSendToShortcut(exe); err != nil {
		return fmt.Errorf("SendTo shortcut: %w", err)
	}
	notifyAssocChanged()
	return nil
}

// InstallClassicVerb keeps the old name used by CLI tests and docs.
func InstallClassicVerb(commandExe string) error {
	return InstallContextMenu(commandExe)
}

// UninstallContextMenu removes classic verbs, leftover QR keys, and the SendTo shortcut.
func UninstallContextMenu() error {
	_ = deleteVerbKeys(oldVerbName)
	_ = deleteVerbKeys(legacyVerb)
	err := deleteVerbKeys(verbName)
	_ = removeSendToShortcut()
	notifyAssocChanged()
	return err
}

// UninstallClassicVerb keeps the old name.
func UninstallClassicVerb() error {
	return UninstallContextMenu()
}

// EnsureContextMenu registers or repairs the menu when the exe path changed
// or hiding values (including ExplorerCommandHandler) are present.
func EnsureContextMenu(commandExe string) error {
	exe, err := currentExe(commandExe)
	if err != nil {
		return err
	}
	if !ShouldAutoRegister(exe) {
		return nil
	}
	_ = deleteVerbKeys(oldVerbName)
	_ = deleteVerbKeys(legacyVerb)
	st := inspectMenu(exe)
	if st.Installed && !st.NeedsRepair {
		return nil
	}
	return InstallContextMenu(exe)
}

// ContextMenuStatus describes the current HKCU verb + SendTo shortcut.
func ContextMenuStatus() MenuStatus {
	exe, err := currentExe("")
	if err != nil {
		st := MenuStatus{Win11Note: win11SettingsNote(), Detail: err.Error()}
		st.Label = menuLabel(st)
		return st
	}
	st := inspectMenu(exe)
	st.Label = menuLabel(st)
	st.Win11Note = win11SettingsNote()
	return st
}

func inspectMenu(exe string) MenuStatus {
	st := MenuStatus{CurrentExe: exe, Win11Note: win11SettingsNote()}
	var hiding []string
	var cmds []string
	for _, keyPath := range verbTargets(verbName) {
		c, hide, err := readVerb(keyPath)
		if err != nil {
			st.Installed = false
			st.NeedsRepair = true
			st.Detail = "missing " + keyPath
			st.SendToOK = sendToShortcutOK(exe)
			st.Label = menuLabel(st)
			return st
		}
		cmds = append(cmds, c)
		hiding = append(hiding, hide...)
	}
	st.Installed = true
	st.RegisteredExe = parseExeFromCommand(cmds[0])
	if !sameExePath(st.RegisteredExe, exe) {
		st.NeedsRepair = true
		st.Detail = "registered exe path differs from this app"
	}
	wantCmd := verbCommand(exe)
	for _, c := range cmds {
		if c != wantCmd {
			st.NeedsRepair = true
			if st.Detail == "" {
				st.Detail = "command line is not the current exe + \"%1\""
			}
		}
	}
	if len(hiding) > 0 {
		st.NeedsRepair = true
		st.Detail = "hiding values: " + strings.Join(hiding, ", ")
	}
	st.SendToOK = sendToShortcutOK(exe)
	if !st.SendToOK {
		st.NeedsRepair = true
		if st.Detail == "" {
			st.Detail = "SendTo shortcut missing or stale"
		}
	}
	st.Label = menuLabel(st)
	return st
}

func readVerb(keyPath string) (command string, hiding []string, err error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", nil, err
	}
	defer k.Close()
	for _, name := range hidingValueNames {
		if _, _, e := k.GetStringValue(name); e == nil {
			hiding = append(hiding, name)
		}
	}
	ck, err := registry.OpenKey(registry.CURRENT_USER, keyPath+`\command`, registry.QUERY_VALUE)
	if err != nil {
		return "", hiding, err
	}
	defer ck.Close()
	command, _, err = ck.GetStringValue("")
	return command, hiding, err
}

func writeVerb(keyPath, exe string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("create %s: %w", keyPath, err)
	}
	defer k.Close()
	if err := k.SetStringValue("", verbLabel); err != nil {
		return err
	}
	if err := k.SetStringValue("MUIVerb", verbLabel); err != nil {
		return err
	}
	if err := k.SetStringValue("Icon", verbIcon(exe)); err != nil {
		return err
	}
	for _, name := range hidingValueNames {
		_ = k.DeleteValue(name)
	}

	ck, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath+`\command`, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}
	defer ck.Close()
	return ck.SetStringValue("", verbCommand(exe))
}

func deleteVerbKeys(name string) error {
	var first error
	for _, keyPath := range verbTargets(name) {
		_ = registry.DeleteKey(registry.CURRENT_USER, keyPath+`\command`)
		if err := registry.DeleteKey(registry.CURRENT_USER, keyPath); err != nil {
			if err != registry.ErrNotExist && first == nil {
				first = err
			}
		}
	}
	return first
}
