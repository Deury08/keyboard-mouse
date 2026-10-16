package main

import (
	"unsafe"
)

// Leyenda en pantalla que aparece al activar el modo cursor, para no tener que
// acordarse de las teclas cada vez.
//
// Es una ventana layered como la cruceta: dibujada en un DIB 32bpp con alfa,
// se actualiza con UpdateLayeredWindow. El borde es transparente y el fondo
// tiene alfa alto, asi que los clicks pasan en el margen y se pueden usar el
// mouse recien activado sin que la leyenda estorbe.

const (
	legendClass = "KbMouseLegendGo"
	legendW     = 620
	legendH     = 300
	legendTTL   = 9 // segundos que queda visible

	// fondo oscuro con algo de transparencia, texto casi blanco
	legendBg     = 0xE61A1A1E // BGRA: alfa E6, B=1A, G=1A, R=1E
	legendFg     = 0x00F2F2F5
	legendAccent = 0x00FFD166
)

var (
	legendWnd   uintptr
	legendProc  uintptr
	legendDC    uintptr
	legendBmpDC uintptr
	legendBmp   uintptr
	legendBits  *uint32

	legendVisible bool
	legendHideAt  int64 // unix nanos
)

// legendLines es el texto de la leyenda.
var legendLines = []string{
	"MOUSE POR TECLADO - modo cursor activo",
	"",
	"Ctrl+Alt+M  volver al teclado        Esc  salir del modo cursor",
	"Tab  cambiar las flechas entre PUNTERO y SCROLL + ZOOM",
	"Flechas  " + arrowModeLinePointer,
	"",
	"Espacio  click          Shift+Espacio  ampliar seleccion",
	"Ctrl+Espacio  click derecho (menu)   Alt+Espacio  click medio",
	"Espacio 2-3 veces  doble o triple click",
	"Espacio + mover  arrastrar",
	"F (pisado) turbo   Z (pisado) 1 px   + y - velocidad",
	"RePag / AvPag  scroll    Ctrl+Alt+Inicio o Fin  esquina",
}

const arrowModeLinePointer = "mueven el puntero"
const arrowModeLineScroll = "arriba/abajo scroll, izquierda/derecha zoom"

// arrowModeLines devuelve el texto con el modo de flechas actual.
func arrowModeLines() []string {
	lines := make([]string, len(legendLines))
	copy(lines, legendLines)
	for i, l := range lines {
		if l == "Flechas  "+arrowModeLinePointer {
			if st.arrowScroll {
				lines[i] = "Flechas  " + arrowModeLineScroll
			}
			break
		}
	}
	return lines
}

func showLegend() {
	if !st.showLegend || legendVisible {
		return
	}
	if !ensureLegendWindow() {
		return
	}
	drawLegend()
	positionLegend()
	pShowWindow.Call(legendWnd, swShow)
	legendVisible = true
	legendHideAt = nowNano() + int64(legendTTL)*1e9
	vis, _, _ := pIsWindowVisible.Call(legendWnd)
	logf("leyenda mostrada por %d s (visible=%d)", legendTTL, vis)
}

func hideLegend() {
	if !legendVisible {
		return
	}
	legendVisible = false
	pShowWindow.Call(legendWnd, swHide)
	logf("leyenda oculta")
}

// legendActive avisa si hay que apagarla por tiempo.
func legendActive() bool {
	if !legendVisible {
		return false
	}
	if nowNano() >= legendHideAt {
		hideLegend()
		return false
	}
	return true
}

func ensureLegendWindow() bool {
	if legendWnd != 0 {
		return true
	}

	legendProc = newCallback(legendWndProc)
	className := utf16Ptr(legendClass)
	wc := wndClassex{
		Size:          uint32(unsafe.Sizeof(wndClassex{})),
		LpfnWndProc:   legendProc,
		HInstance:     moduleHandle(),
		LpszClassName: className,
	}
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	legendWnd, _, _ = pCreateWindowEx.Call(
		uintptr(wsExTopmost|wsExToolWin|wsExNoActivate|wsExLayered),
		uintptr(unsafe.Pointer(className)), 0,
		uintptr(wsPopup),
		0, 0, uintptr(legendW), uintptr(legendH),
		0, 0, moduleHandle(), 0)
	if legendWnd == 0 {
		logf("ERROR: no se pudo crear la ventana de la leyenda")
		return false
	}

	legendDC, _, _ = pGetDC.Call(0)
	legendBmpDC, _, _ = pCreateCompatibleDC.Call(legendDC)

	var bmi bitmapInfo
	bmi.Header.Size = uint32(unsafe.Sizeof(bitmapInfoHeader{}))
	bmi.Header.Width = legendW
	bmi.Header.Height = -legendH
	bmi.Header.Planes = 1
	bmi.Header.BitCount = 32
	bmi.Header.Compression = 0

	hbmp, _, _ := pCreateDIBSection.Call(legendDC, uintptr(unsafe.Pointer(&bmi)), 0,
		uintptr(unsafe.Pointer(&legendBits)), 0, 0)
	if hbmp == 0 || legendBits == nil {
		logf("ERROR: no se pudo crear la superficie de la leyenda")
		return false
	}
	legendBmp = hbmp
	pSelectObject.Call(legendBmpDC, legendBmp)
	return true
}

// drawLegend pinta el fondo y escribe el texto con GDI sobre el DIB.
func drawLegend() {
	if legendBits == nil {
		return
	}
	pix := unsafe.Slice(legendBits, legendW*legendH)
	// fondo: opaco en el medio, transparente en los bordes de 1 px
	const margin = 2
	for y := 0; y < legendH; y++ {
		for x := 0; x < legendW; x++ {
			i := y*legendW + x
			edge := x < margin || y < margin || x >= legendW-margin || y >= legendH-margin
			if edge {
				pix[i] = 0
			} else {
				pix[i] = legendBg
			}
		}
	}

	if !selectLegendFont() {
		return
	}
	pSetBkMode.Call(legendBmpDC, 1 /* TRANSPARENT */)

	y := 14
	for i, line := range arrowModeLines() {
		if line == "" {
			y += 14
			continue
		}
		color := uintptr(legendFg)
		if i == 0 {
			color = legendAccent
		}
		pSetTextColor.Call(legendBmpDC, color)
		lineUTF16, _ := syscallUTF16(line)
		// TextOutW y no DrawTextW: DrawTextW se cae con este puente de syscall
		pTextOut.Call(legendBmpDC, 16, uintptr(y),
			uintptr(unsafe.Pointer(&lineUTF16[0])), uintptr(len(lineUTF16)))
		y += 28
	}
}

var legendFont uintptr

func selectLegendFont() bool {
	if legendFont != 0 {
		pSelectObject.Call(legendBmpDC, legendFont)
		return true
	}
	face, _ := syscallUTF16("Segoe UI")
	if len(face) == 0 {
		return false
	}
	// CreateFontW pide 14 parametros: alto, ancho, escape, orientacion, peso,
	// cursiva, subrayado, tachado, charset, precision salida, precision clip,
	// calidad, pitchAndFamily y por ultimo el nombre de la fuente.
	legendFont, _, _ = pCreateFont.Call(
		22, 0, 0, 0, 400, 0, 0, 0,
		1 /* DEFAULT_CHARSET */, 0, 0, 0,
		0 /* DEFAULT_PITCH */, uintptr(unsafe.Pointer(&face[0])))
	if legendFont == 0 {
		legendFont, _, _ = pGetStockObject.Call(defaultGuiFont)
	}
	if legendFont == 0 {
		logf("ERROR: no se pudo crear la fuente de la leyenda")
		return false
	}
	pSelectObject.Call(legendBmpDC, legendFont)
	return true
}

// positionLegend la centra arriba de la pantalla, en el monitor principal.
func positionLegend() {
	x := int32((systemMetrics(0) - legendW) / 2)
	y := int32(40)
	pos := point{X: x, Y: y}
	size := size{Cx: legendW, Cy: legendH}
	src := point{}
	blend := layeredBlend{BlendOp: acSrcOver, AlphaFormat: acSrcAlpha, SourceConstantAlpha: 255}
	ok, _, _ := pUpdateLayeredWindow.Call(
		legendWnd, legendDC,
		uintptr(unsafe.Pointer(&pos)), uintptr(unsafe.Pointer(&size)),
		legendBmpDC, uintptr(unsafe.Pointer(&src)),
		0, uintptr(unsafe.Pointer(&blend)), ulwAlpha)
	if st.verbose {
		vis, _, _ := pIsWindowVisible.Call(legendWnd)
		logf("leyenda: UpdateLayeredWindow=%d visible=%d error=%d", ok, vis, lastError())
	}
}

func legendWndProc(hwnd uintptr, m uint32, wParam, lParam uintptr) uintptr {
	switch m {
	case wmNCHitTest:
		return ^uintptr(0) // HTTRANSPARENT: no robe los clicks
	case wmPaint:
		return 0
	case wmDestroy:
		legendWnd = 0
		legendVisible = false
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, uintptr(m), wParam, lParam)
	return r
}
