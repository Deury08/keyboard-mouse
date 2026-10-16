package main

import "unsafe"

// Cruceta roja que sigue al cursor.
//
// Importa: es una ventana layered con alfa por pixel. Solo los pixeles del
// drawing tienen alfa 255; el resto es transparente. Windows deja pasar los
// clicks a la ventana de abajo donde el alfa es 0, que es lo unico que
// funciona de verdad: una ventana normal (WS_POPUP) encima del puntero se
// queda con todos los clicks y el mouse virtual deja de hacer nada.

const (
	indicatorClass = "KbMouseCrosshairGo"
	indicatorSize  = 41
	indicatorArm   = 12 // largo de cada brazo de la cruz, en pixeles

	crossColor = 0x000000FF // BGRA premultiplicado: rojo opaco
)

var (
	indicatorWnd   uintptr
	indicatorProc  uintptr
	indicatorDC    uintptr
	indicatorBmp   uintptr
	indicatorBmpDC uintptr
	indicatorBits  *uint32
)

func showIndicator() {
	if !st.showCross || indicatorWnd != 0 {
		return
	}

	indicatorProc = newCallback(indicatorWndProc)
	className := utf16Ptr(indicatorClass)
	wc := wndClassex{
		Size:          uint32(unsafe.Sizeof(wndClassex{})),
		LpfnWndProc:   indicatorProc,
		HInstance:     moduleHandle(),
		LpszClassName: className,
	}
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	indicatorWnd, _, _ = pCreateWindowEx.Call(
		uintptr(wsExTopmost|wsExToolWin|wsExNoActivate|wsExLayered),
		uintptr(unsafe.Pointer(className)), 0,
		uintptr(wsPopup),
		0, 0, uintptr(indicatorSize), uintptr(indicatorSize),
		0, 0, moduleHandle(), 0)
	if indicatorWnd == 0 {
		logf("ERROR: no se pudo crear la ventana de la cruceta")
		return
	}

	if !buildIndicatorSurface() {
		logf("ERROR: no se pudo crear la superficie de la cruceta")
		return
	}
	followCursor()
	// UpdateLayeredWindow no deja la ventana visible por si solo: hay que
	// mostrarla una vez, si no la cruceta existe pero no se ve.
	pShowWindow.Call(indicatorWnd, swShow)

	vis, _, _ := pIsWindowVisible.Call(indicatorWnd)
	logf("cruceta creada (click-through, layered): hwnd=%d visible=%d en %d,%d",
		indicatorWnd, vis, cursorX(), cursorY())
}

// buildIndicatorSurface prepara el DIB 32bpp donde se dibuja la cruz.
func buildIndicatorSurface() bool {
	indicatorDC, _, _ = pGetDC.Call(0) // DC de pantalla
	if indicatorDC == 0 {
		return false
	}
	indicatorBmpDC, _, _ = pCreateCompatibleDC.Call(indicatorDC)
	if indicatorBmpDC == 0 {
		return false
	}

	var bmi bitmapInfo
	bmi.Header.Size = uint32(unsafe.Sizeof(bitmapInfoHeader{}))
	bmi.Header.Width = indicatorSize
	bmi.Header.Height = -indicatorSize // negativo: de arriba hacia abajo
	bmi.Header.Planes = 1
	bmi.Header.BitCount = 32
	bmi.Header.Compression = 0 // BI_RGB

	hbmp, _, _ := pCreateDIBSection.Call(indicatorDC, uintptr(unsafe.Pointer(&bmi)),
		0 /* DIB_RGB_COLORS */, uintptr(unsafe.Pointer(&indicatorBits)), 0, 0)
	if hbmp == 0 || indicatorBits == nil {
		return false
	}
	indicatorBmp = hbmp
	pSelectObject.Call(indicatorBmpDC, indicatorBmp)

	drawCross()
	return true
}

// drawCross pinta la cruz en el DIB: fondo totalmente transparente (alfa 0) y
// solo los dos brazos con color.
func drawCross() {
	if indicatorBits == nil {
		return
	}
	pix := unsafe.Slice(indicatorBits, indicatorSize*indicatorSize)
	for i := range pix {
		pix[i] = 0 // alfa 0 = transparente y sin clicks
	}
	c := indicatorSize / 2
	for i := -indicatorArm; i <= indicatorArm; i++ {
		for w := 0; w < 2; w++ { // 2 px de grosor
			h := c + w
			setPixel(c+i, h)
			setPixel(h, c+i)
		}
	}
}

func setPixel(x, y int) {
	if x < 0 || y < 0 || x >= indicatorSize || y >= indicatorSize {
		return
	}
	unsafe.Slice(indicatorBits, indicatorSize*indicatorSize)[y*indicatorSize+x] = crossColor
}

func hideIndicator() {
	if indicatorWnd == 0 {
		return
	}
	logf("cruceta oculta")
	pDestroyWindow.Call(indicatorWnd)
	indicatorWnd = 0

	if indicatorBmpDC != 0 {
		pDeleteDC.Call(indicatorBmpDC)
		indicatorBmpDC = 0
	}
	if indicatorBmp != 0 {
		pDeleteObject.Call(indicatorBmp)
		indicatorBmp = 0
	}
	if indicatorDC != 0 {
		pReleaseDC.Call(0, indicatorDC)
		indicatorDC = 0
	}
	indicatorBits = nil
}

// followCursor mueve la cruceta al cursor actual.
func followCursor() {
	if indicatorWnd == 0 {
		return
	}
	p, ok := cursorPos()
	if !ok {
		return
	}

	pos := point{X: p.X - indicatorSize/2, Y: p.Y - indicatorSize/2}
	size := size{Cx: indicatorSize, Cy: indicatorSize}
	src := point{X: 0, Y: 0}
	blend := layeredBlend{BlendOp: acSrcOver, AlphaFormat: acSrcAlpha, SourceConstantAlpha: 255}

	ok2, _, _ := pUpdateLayeredWindow.Call(
		indicatorWnd, indicatorDC,
		uintptr(unsafe.Pointer(&pos)), uintptr(unsafe.Pointer(&size)),
		indicatorBmpDC, uintptr(unsafe.Pointer(&src)),
		0, uintptr(unsafe.Pointer(&blend)), ulwAlpha)

	if st.verbose {
		vis, _, _ := pIsWindowVisible.Call(indicatorWnd)
		cx, _, _ := pGetWindowLong.Call(indicatorWnd, ^uintptr(19)) // GWL_EXSTYLE = -20 // GWL_EXSTYLE
		logf("UpdateLayeredWindow=%d error=%d visible=%d exstyle=0x%X bits=%p",
			ok2, lastError(), vis, cx, indicatorBits)
	}
	if ok2 == 0 {
		logf("ERROR: UpdateLayeredWindow fallo (error %d)", lastError())
	}
}

func indicatorWndProc(hwnd uintptr, m uint32, wParam, lParam uintptr) uintptr {
	switch m {
	case wmNCHitTest:
		// extra de seguridad: si algo llega, se reenvia a la ventana de abajo
		return ^uintptr(0) // (LONG)-1 = HTTRANSPARENT
	case wmPaint:
		// una ventana layered no necesita pintar: el contenido ya esta en el DIB
		return 0
	case wmDestroy:
		indicatorWnd = 0
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, uintptr(m), wParam, lParam)
	return r
}

func lastError() uint32 {
	r, _, _ := pGetLastError.Call()
	return uint32(r)
}

func cursorX() int32 {
	p, _ := cursorPos()
	return p.X
}

func cursorY() int32 {
	p, _ := cursorPos()
	return p.Y
}
