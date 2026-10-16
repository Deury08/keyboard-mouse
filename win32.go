package main

import (
	"syscall"
	"unsafe"
)

// DLLs y Handles de Windows. Todo con la libreria estandar: sin dependencias
// externas, sin go.mod con modulos, para que el .exe sea autocontenido.
var (
	user32  = syscall.NewLazyDLL("user32.dll")
	gdi32   = syscall.NewLazyDLL("gdi32.dll")
	shell32 = syscall.NewLazyDLL("shell32.dll")
)

const (
	whKeyboardLL = 13
	hcAction     = 0

	ulwAlpha   = 0x02
	acSrcOver  = 0x00
	acSrcAlpha = 0x01
)

const (
	wmDestroy    = 0x0002
	wmPaint      = 0x000D
	wmEraseBkgnd = 0x0014
	wmCommand    = 0x0111
	wmKeyDown    = 0x0100
	wmSysKeyDown = 0x0104
	wmKeyUp      = 0x0101
	wmSysKeyUp   = 0x0105
	wmNCHitTest  = 0x0084

	defaultGuiFont = 17 // DEFAULT_GUI_FONT
	wmApp          = 0x8000
	wmTray         = wmApp + 1
	wmNull         = 0x0000
)

const (
	wsPopup         = 0x80000000
	wsVisible       = 0x10000000
	wsExTopmost     = 0x00000008
	wsExToolWin     = 0x00000080
	wsExNoActivate  = 0x08000000
	wsExTransparent = 0x00000020
	wsExLayered     = 0x00080000
	csOwnDC         = 0x0020
	swShow          = 5
	swHide          = 0
)

const (
	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010
	swpShowWindow = 0x0040
	hwMessage     = ^uintptr(2) // (HWND)-3, ventana que solo recibe mensajes
)

// Virtual keys usados.
const (
	vkShift    = 0x10
	vkControl  = 0x11
	vkMenu     = 0x12
	vkPause    = 0x13
	vkEscape   = 0x1B
	vkOemPlus  = 0xBB // la tecla + del teclado numerico
	vkOemMinus = 0xBD // la tecla - del teclado numerico
	vkSpace    = 0x20
	vkPrior    = 0x21
	vkNext     = 0x22
	vkEnd      = 0x23
	vkHome     = 0x24
	vkLeft     = 0x25
	vkUp       = 0x26
	vkRight    = 0x27
	vkDown     = 0x28
	vkKeyA     = 0x41
	vkKeyD     = 0x44
	vkKeyH     = 0x48
	vkKeyF     = 0x46
	vkKeyM     = 0x4D
	vkKeyZ     = 0x5A
	vkKeyQ     = 0x51
	vkKeyS     = 0x53
	vkKeyW     = 0x57
)

// Structs de Win32.
type (
	point struct{ X, Y int32 }

	rect struct{ Left, Top, Right, Bottom int32 }

	kbdllhookstruct struct {
		VkCode      uint32
		ScanCode    uint32
		Flags       uint32
		Time        uint32
		DwExtraInfo uintptr
	}

	mouseInput struct {
		Dx, Dy      int32
		MouseData   uint32
		DwFlags     uint32
		Time        uint32
		DwExtraInfo uintptr
	}

	// INPUT con la union explicita: solo usamos la parte de mouse.
	input struct {
		Type uint32
		_    uint32
		Mi   mouseInput
	}

	size struct{ Cx, Cy int32 }

	bitmapInfoHeader struct {
		Size          uint32
		Width         int32
		Height        int32
		Planes        uint16
		BitCount      uint16
		Compression   uint32
		SizeImage     uint32
		XPelsPerMeter int32
		YPelsPerMeter int32
		ClrUsed       uint32
		ClrImportant  uint32
	}

	bitmapInfo struct {
		Header bitmapInfoHeader
		Colors [1]uint32
	}

	layeredBlend struct {
		BlendOp             byte
		BlendFlags          byte
		SourceConstantAlpha byte
		AlphaFormat         byte
	}

	wndClassex struct {
		Size          uint32
		Style         uint32
		LpfnWndProc   uintptr
		CbClsExtra    int32
		CbWndExtra    int32
		HInstance     uintptr
		HIcon         uintptr
		HCursor       uintptr
		HbrBackground uintptr
		LpszMenuName  *uint16
		LpszClassName *uint16
		HIconSm       uintptr
	}

	msg struct {
		Hwnd    uintptr
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		PtX     int32
		PtY     int32
	}

	paintStruct struct {
		Hdc         uintptr
		Erase       int32
		RcPaint     rect
		Restore     int32
		IncUpdate   int32
		RgbReserved [32]byte
	}

	notifyIconData struct {
		CbSize           uint32
		Hwnd             uintptr
		UID              uint32
		UFlags           uint32
		UCallbackMessage uint32
		HIcon            uintptr
		SzTip            [128]uint16
		DwState          uint32
		DwStateMask      uint32
		SzInfo           [256]uint16
		UVersion         uint32
		SzInfoTitle      [64]uint16
		DwInfoFlags      uint32
		GuidItem         [16]byte
		HBalloonIcon     uintptr
	}
)

// Procs: se resuelven en el primer uso (NewLazyDLL es lazy).
var (
	pSetWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	pUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	pCallNextHookEx      = user32.NewProc("CallNextHookEx")
	pGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	pSendInput           = user32.NewProc("SendInput")
	pGetCursorPos        = user32.NewProc("GetCursorPos")
	pSetCursorPos        = user32.NewProc("SetCursorPos")
	pGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	pGetModuleHandle     = kernel32.NewProc("GetModuleHandleW")
	pRegisterClassEx     = user32.NewProc("RegisterClassExW")
	pCreateWindowEx      = user32.NewProc("CreateWindowExW")
	pDestroyWindow       = user32.NewProc("DestroyWindow")
	pSetWindowPos        = user32.NewProc("SetWindowPos")
	pShowWindow          = user32.NewProc("ShowWindow")
	pDefWindowProc       = user32.NewProc("DefWindowProcW")
	pBeginPaint          = user32.NewProc("BeginPaint")
	pEndPaint            = user32.NewProc("EndPaint")
	pGetDC               = user32.NewProc("GetDC")
	pReleaseDC           = user32.NewProc("ReleaseDC")
	pUpdateLayeredWindow = user32.NewProc("UpdateLayeredWindow")
	pCreatePen           = gdi32.NewProc("CreatePen")
	pCreateCompatibleDC  = gdi32.NewProc("CreateCompatibleDC")
	pDeleteDC            = gdi32.NewProc("DeleteDC")
	pCreateDIBSection    = gdi32.NewProc("CreateDIBSection")
	pCreateFont          = gdi32.NewProc("CreateFontW")
	pSetTextColor        = gdi32.NewProc("SetTextColor")
	pSetBkMode           = gdi32.NewProc("SetBkMode")
	pTextOut             = gdi32.NewProc("TextOutW")
	pGetStockObject      = gdi32.NewProc("GetStockObject")
	pGetLastError        = kernel32.NewProc("GetLastError")
	pIsWindowVisible     = user32.NewProc("IsWindowVisible")
	pGetWindowLong       = user32.NewProc("GetWindowLongPtrW")
	pSelectObject        = gdi32.NewProc("SelectObject")
	pDeleteObject        = gdi32.NewProc("DeleteObject")
	pMoveToEx            = gdi32.NewProc("MoveToEx")
	pLineTo              = gdi32.NewProc("LineTo")
	pShellNotifyIcon     = shell32.NewProc("Shell_NotifyIconW")
	pCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	pAppendMenuW         = user32.NewProc("AppendMenuW")
	pTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	pDestroyMenu         = user32.NewProc("DestroyMenu")
	pSetForegroundWnd    = user32.NewProc("SetForegroundWindow")
	pMessageBoxW         = user32.NewProc("MessageBoxW")
	pGetMessageW         = user32.NewProc("GetMessageW")
	pTranslateMessage    = user32.NewProc("TranslateMessage")
	pDispatchMessageW    = user32.NewProc("DispatchMessageW")
	pPostQuitMessage     = user32.NewProc("PostQuitMessage")
	pPostMessage         = user32.NewProc("PostMessageW")

	pCreateMutexW       = kernel32.NewProc("CreateMutexW")
	pCloseHandle        = kernel32.NewProc("CloseHandle")
	pGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
)

// kernel32 no estaba declarado arriba para no repetir el alias.
var kernel32 = syscall.NewLazyDLL("kernel32.dll")

func utf16Ptr(s string) *uint16 { return syscall.StringToUTF16Ptr(s) }

func setWindowsHookEx(idHook int32, lpfn uintptr, hMod uintptr, tid uint32) (uintptr, bool) {
	r, _, _ := pSetWindowsHookEx.Call(uintptr(idHook), lpfn, hMod, uintptr(tid))
	return r, r != 0
}

func callNextHookEx(hhk uintptr, nCode int32, wParam, lParam uintptr) uintptr {
	r, _, _ := pCallNextHookEx.Call(hhk, uintptr(nCode), wParam, lParam)
	return r
}

func keyDown(vk int32) bool {
	r, _, _ := pGetAsyncKeyState.Call(uintptr(vk))
	return uint16(r)&0x8000 != 0
}

func modMask() int {
	m := 0
	if keyDown(vkControl) {
		m |= 1
	}
	if keyDown(vkMenu) {
		m |= 2
	}
	if keyDown(vkShift) {
		m |= 4
	}
	return m
}

func sendInput(inputs []input) uint32 {
	if len(inputs) == 0 {
		return 0
	}
	r, _, _ := pSendInput.Call(uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		uintptr(unsafe.Sizeof(inputs[0])))
	return uint32(r)
}

func cursorPos() (point, bool) {
	var p point
	r, _, _ := pGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p, r != 0
}

func setCursorPos(x, y int32) bool {
	r, _, _ := pSetCursorPos.Call(uintptr(x), uintptr(y))
	return r != 0
}

func systemMetrics(index int32) int32 {
	r, _, _ := pGetSystemMetrics.Call(uintptr(index))
	return int32(r)
}

func moduleHandle() uintptr {
	r, _, _ := pGetModuleHandle.Call(0)
	return r
}

func messageBox(hwnd uintptr, text, caption string, flags uintptr) {
	pMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(unsafe.Pointer(utf16Ptr(caption))), flags)
}

func getCurrentThreadId() uint32 {
	r, _, _ := pGetCurrentThreadId.Call()
	return uint32(r)
}

// syscallUTF16 es un atajo para no importar syscall en todos lados.
func syscallUTF16(s string) ([]uint16, error) { return syscall.UTF16FromString(s) }
