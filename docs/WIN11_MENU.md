# Windows 11 modern context menu (C1 / M4.7)

## What “top-level” means

Windows 11’s compact right-click menu only shows verbs that come from a **packaged `IExplorerCommand`** COM object (sparse / MSIX package with identity). Classic `HKCU\...\shell\AirLink` verbs appear under **Show more options** (and on Windows 10’s full menu).

## What this repo installs today

First run of the Wails exe (or `--install-menu`) writes the **classic HKCU verb** (files + folders) and a **Send to** shortcut. It **does not** set `ExplorerCommandHandler` on that verb.

The compact-menu sparse package under `internal/shell/win11/` is a **later** task: it needs a signed identity. Putting `ExplorerCommandHandler` on the classic verb without that identity made Explorer hide **Send with Air-Link** even under Show more options.

## If the compact item is missing

Expected on Windows 11 until a signed package exists. Use **Show more options** or Shift+right-click, or **Send to → Send with Air-Link**.

## Windows 10

Classic HKCU verb on the full context menu.
