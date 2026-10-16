package main

import (
	"syscall"
	"unsafe"
)

// Bandeja del sistema: la app se ve solo por el icono, sin ventanas de terminal.
const (
	nimAdd    = 0
	nimDelete = 2
	nimModify = 1

	nifMessage = 0x01
	nifIcon    = 0x02
	nifTip     = 0x04
	nifInfo    = 0x10

	wmLButtonUp = 0x0202
	wmRButtonUp = 0x0205

	tpmReturnCmd   = 0x0100
	tpmRightButton = 0x0002

	mfString = 0x0000

	menuToggle = 1001
	menuHelp   = 1002
	menuExit   = 1003

	trayUID = 1
)

var (
	trayWnd     uintptr
	trayProc    uintptr
	trayIcon    uintptr
	trayPresent bool
)

func showTray() {
	trayProc = newCallback(trayWndProc)
	className := utf16Ptr("KbMouseTrayGo")
	wc := wndClassex{
		Size:          uint32(unsafe.Sizeof(wndClassex{})),
		LpfnWndProc:   trayProc,
		HInstance:     moduleHandle(),
		LpszClassName: className,
	}
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	// ventana invisible que solo recibe mensajes
	trayWnd, _, _ = pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(className)), 0, 0,
		0, 0, 0, 0, uintptr(hwMessage), 0, moduleHandle(), 0)

	trayIcon, _, _ = user32.NewProc("LoadIconW").Call(0, 32512 /* IDI_APPLICATION */)

	nid := notifyIconData{
		CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		Hwnd:             trayWnd,
		UID:              trayUID,
		UFlags:           nifMessage | nifIcon | nifTip,
		UCallbackMessage: wmTray,
		HIcon:            trayIcon,
	}
	copy(nid.SzTip[:], utf16("Mouse por teclado"))
	if addTrayIcon(&nid) {
		trayPresent = true
	}
}

func removeTray() {
	if !trayPresent {
		return
	}
	nid := notifyIconData{CbSize: uint32(unsafe.Sizeof(notifyIconData{})), Hwnd: trayWnd, UID: trayUID}
	pShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	trayPresent = false
}

// updateTray refleja el estado actual en el tooltip del icono.
func updateTray() {
	if !trayPresent {
		return
	}
	tip := "Mouse por teclado - modo cursor OFF"
	if st.cursorMode {
		tip = "Mouse por teclado - modo cursor ON"
	}
	nid := notifyIconData{
		CbSize: uint32(unsafe.Sizeof(notifyIconData{})),
		Hwnd:   trayWnd,
		UID:    trayUID,
		UFlags: nifTip,
	}
	copy(nid.SzTip[:], utf16(tip))
	addTrayIcon(&nid)
}

func addTrayIcon(nid *notifyIconData) bool {
	r, _, _ := pShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(nid)))
	if r == 0 {
		// NIM_MODIFY usa la misma estructura con NIM_MODIFY
		r, _, _ = pShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(nid)))
	}
	return r != 0
}

func showTrayMenu() {
	menu, _, _ := pCreatePopupMenu.Call()
	label := "Activar modo cursor"
	if st.cursorMode {
		label = "Desactivar modo cursor"
	}
	pAppendMenuW.Call(menu, mfString, menuToggle, uintptr(unsafe.Pointer(utf16Ptr(label))))
	pAppendMenuW.Call(menu, mfString, menuHelp, uintptr(unsafe.Pointer(utf16Ptr("Ayuda (Ctrl+Alt+H)"))))
	pAppendMenuW.Call(menu, mfString, menuExit, uintptr(unsafe.Pointer(utf16Ptr("Salir"))))

	p, _ := cursorPos()
	pSetForegroundWnd.Call(trayWnd)
	cmd, _, _ := pTrackPopupMenu.Call(menu, uintptr(tpmReturnCmd|tpmRightButton), uintptr(p.X), uintptr(p.Y), 0, trayWnd, 0)
	// PostMessage(WM_NULL) es el truco documentado para cerrar el menu en
	// Windows; PostQuitMessage aqui mataria la app.
	pPostMessage.Call(trayWnd, wmNull, 0, 0)
	pDestroyMenu.Call(menu)

	switch cmd {
	case menuToggle:
		toggleCursorMode()
	case menuHelp:
		showHelp()
	case menuExit:
		quitApp()
	}
}

func trayWndProc(hwnd uintptr, m uint32, wParam, lParam uintptr) uintptr {
	switch m {
	case wmTray:
		switch lParam {
		case wmLButtonUp:
			toggleCursorMode()
		case wmRButtonUp:
			showTrayMenu()
		}
		return 0
	case wmCommand:
		switch wParam & 0xFFFF {
		case menuToggle:
			toggleCursorMode()
		case menuHelp:
			showHelp()
		case menuExit:
			quitApp()
		}
		return 0
	case wmDestroy:
		removeTray()
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, uintptr(m), wParam, lParam)
	return r
}

// newCallback crea un callback de Win32 y lo mantiene vivo.
func newCallback(fn any) uintptr {
	return syscall.NewCallback(fn)
}

func utf16(s string) []uint16 {
	ptr, err := syscall.UTF16FromString(s)
	if err != nil {
		return nil
	}
	return ptr
}

// showWelcomeBalloon manda un aviso flotante al iniciar: es la unica señal
// visible de que la app arranco y quedo instalada.
func showWelcomeBalloon() {
	nid := notifyIconData{
		CbSize:      uint32(unsafe.Sizeof(notifyIconData{})),
		Hwnd:        trayWnd,
		UID:         trayUID,
		UFlags:      nifInfo,
		DwInfoFlags: 0x01, // NIIF_INFO
		HIcon:       trayIcon,
	}
	copy(nid.SzInfoTitle[:], utf16("Mouse por teclado"))
	copy(nid.SzInfo[:], utf16("Listo. Ctrl+Alt+M cambia teclado <-> mouse."))
	r, _, _ := pShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
	debugf("balloon de bienvenida: %v", r != 0)
}
