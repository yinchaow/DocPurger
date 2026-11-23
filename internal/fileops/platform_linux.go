//go:build linux

package fileops

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RevealInFolder opens the file manager and highlights the target file on Kylin V10 SP1 (UKUI Peony) / Linux.
func RevealInFolder(targetPath string) error {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	// 1. Prefer Kylin V10 SP1 UKUI Peony file manager (--show-items highlights the file)
	if peony, err := exec.LookPath("peony"); err == nil {
		cmd := exec.Command(peony, "--show-items", absPath)
		if cmd.Start() == nil {
			go func() { _ = cmd.Wait() }()
			return nil
		}
	}

	// 2. Standard FreeDesktop FileManager1 D-Bus ShowItems (supported by Peony, Nautilus, Dolphin, Nemo, Caja)
	if dbusSend, err := exec.LookPath("dbus-send"); err == nil {
		fileURI := "file://" + escapeFileURI(absPath)
		cmd := exec.Command(
			dbusSend,
			"--session",
			"--dest=org.freedesktop.FileManager1",
			"--type=method_call",
			"/org/freedesktop/FileManager1",
			"org.freedesktop.FileManager1.ShowItems",
			"array:string:"+fileURI,
			"string:",
		)
		if cmd.Run() == nil {
			return nil
		}
	}

	// 3. Fallback: open parent directory with xdg-open
	parent := filepath.Dir(absPath)
	cmd := exec.Command("xdg-open", parent)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// OpenFile opens the document using the system default application.
func OpenFile(targetPath string) error {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}
	cmd := exec.Command("xdg-open", absPath)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// MoveToTrash moves a file into the FreeDesktop XDG Trash (~/.local/share/Trash),
// fully compatible with Kylin V10 SP1 (Peony) trash restore.
func MoveToTrash(targetPath string) error {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absPath); err != nil {
		return err
	}

	// Try `gio trash` first if available (handles mountpoints natively)
	if gio, err := exec.LookPath("gio"); err == nil {
		cmd := exec.Command(gio, "trash", absPath)
		if cmd.Run() == nil {
			return nil
		}
	}

	// Pure Go FreeDesktop.org Trash Spec implementation
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	trashFilesDir := filepath.Join(dataHome, "Trash", "files")
	trashInfoDir := filepath.Join(dataHome, "Trash", "info")
	if err := os.MkdirAll(trashFilesDir, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(trashInfoDir, 0o700); err != nil {
		return err
	}

	baseName := filepath.Base(absPath)
	ext := filepath.Ext(baseName)
	stem := strings.TrimSuffix(baseName, ext)

	var trashName, infoPath, destPath string
	for i := 0; i < 10000; i++ {
		if i == 0 {
			trashName = baseName
		} else {
			trashName = fmt.Sprintf("%s.%d%s", stem, i, ext)
		}
		infoPath = filepath.Join(trashInfoDir, trashName+".trashinfo")
		destPath = filepath.Join(trashFilesDir, trashName)

		// Atomically reserve .trashinfo
		f, err := os.OpenFile(infoPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				continue
			}
			return err
		}
		content := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n",
			escapeFileURI(absPath),
			time.Now().Format("2006-01-02T15:04:05"),
		)
		_, writeErr := f.WriteString(content)
		_ = f.Close()
		if writeErr != nil {
			_ = os.Remove(infoPath)
			return writeErr
		}
		break
	}

	if err := moveFileCrossDevice(absPath, destPath); err != nil {
		_ = os.Remove(infoPath)
		return err
	}
	return nil
}

func escapeFileURI(p string) string {
	parts := strings.Split(filepath.ToSlash(p), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
