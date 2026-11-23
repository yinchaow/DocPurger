//go:build windows

package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	shell32                        = syscall.NewLazyDLL("shell32.dll")
	ole32                          = syscall.NewLazyDLL("ole32.dll")
	procSHFileOperation            = shell32.NewProc("SHFileOperationW")
	procSHOpenFolderAndSelectItems = shell32.NewProc("SHOpenFolderAndSelectItems")
	procILCreateFromPathW          = shell32.NewProc("ILCreateFromPathW")
	procILFree                     = shell32.NewProc("ILFree")
	procShellExecuteW              = shell32.NewProc("ShellExecuteW")
	procCoInitializeEx             = ole32.NewProc("CoInitializeEx")
	procCoUninitialize             = ole32.NewProc("CoUninitialize")
)

const (
	foDelete               = 0x0003
	fofSilent              = 0x0004
	fofNoConfirmation      = 0x0010
	fofAllowUndo           = 0x0040
	fofNoErrorUI           = 0x0400
	coinitApartmentThread  = 0x2
	swShowNormal           = 1
)

type shFileOpStructW struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

func shellExecuteOpen(target string) error {
	verbPtr, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	ret, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(targetPtr)),
		0,
		0,
		swShowNormal,
	)
	if ret <= 32 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return fmt.Errorf("打开失败: %w", callErr)
		}
		return fmt.Errorf("打开失败 (ShellExecute code: %d)", ret)
	}
	return nil
}

// RevealInFolder opens Windows Explorer with the specified file selected using native
// Shell32 COM API (SHOpenFolderAndSelectItems), avoiding spawning explorer.exe child processes.
func RevealInFolder(targetPath string) error {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absPath); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		hrInit, _, _ := procCoInitializeEx.Call(0, coinitApartmentThread)
		if hrInit == 0 || hrInit == 1 { // S_OK or S_FALSE
			defer procCoUninitialize.Call()
		}

		pathPtr, err := syscall.UTF16PtrFromString(absPath)
		if err != nil {
			errCh <- err
			return
		}

		pidl, _, _ := procILCreateFromPathW.Call(uintptr(unsafe.Pointer(pathPtr)))
		if pidl == 0 {
			errCh <- shellExecuteOpen(filepath.Dir(absPath))
			return
		}
		defer procILFree.Call(pidl)

		// Per MSDN: when cidl == 0, pidlFolder can be a file's PIDL to open its parent folder and select it.
		hr, _, _ := procSHOpenFolderAndSelectItems.Call(pidl, 0, 0, 0)
		if int32(hr) < 0 {
			errCh <- shellExecuteOpen(filepath.Dir(absPath))
			return
		}
		errCh <- nil
	}()
	return <-errCh
}

// OpenFile opens the file with its associated default Windows application using native ShellExecuteW.
func OpenFile(targetPath string) error {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}
	return shellExecuteOpen(absPath)
}

// MoveToTrash sends the file to the Windows Recycle Bin using native SHFileOperationW (zero CGO).
func MoveToTrash(targetPath string) error {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absPath); err != nil {
		return err
	}

	// SHFileOperationW requires a double-null-terminated UTF-16 string
	u16, err := syscall.UTF16FromString(absPath)
	if err != nil {
		return err
	}
	u16 = append(u16, 0) // second NUL terminator

	op := shFileOpStructW{
		wFunc:  foDelete,
		pFrom:  &u16[0],
		fFlags: fofAllowUndo | fofNoConfirmation | fofNoErrorUI | fofSilent,
	}

	ret, _, _ := procSHFileOperation.Call(uintptr(unsafe.Pointer(&op)))
	if ret != 0 {
		return fmt.Errorf("移入回收站失败 (SHFileOperation code: 0x%x)", ret)
	}
	if op.fAnyOperationsAborted != 0 {
		return fmt.Errorf("移入回收站操作被中止")
	}
	return nil
}
