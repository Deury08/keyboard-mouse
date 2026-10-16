package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"
)

// Flags de SendInput para el mouse.
const (
	fMove       = 0x0001
	fLeftDown   = 0x0002
	fLeftUp     = 0x0004
	fRightDown  = 0x0008
	fRightUp    = 0x0010
	fMiddleDown = 0x0020
	fMiddleUp   = 0x0040
	fWheel      = 0x0800
	fHWheel     = 0x01000
)

// Botones.
const (
	btnLeft = iota
	btnRight
	btnMiddle
)

type appState struct {
	speed     int
	turbo     int
	autoExit  int
	toggleVk  int32
	toggleMod int

	cursorMode bool
	showCross  bool
	verbose    bool
	freeKeys   bool // en modo cursor, dejar pasar todas las teclas

	lastActivity time.Time

	// maquina de estados del boton: distinguishes tap (click) de hold (drag)
	btnDown     bool
	btnKind     int
	btnDownTime time.Time
	btnDownPos  point
	lastClick   time.Time
	lastCount   int
}

var lastPollLog = time.Time{}

var st = &appState{speed: 12, turbo: 60, autoExit: 45, showCross: true}

// pressedKeys es el estado real de las teclas,Armado por el propio hook: cada
// keydown/keyup entra por el hook y actualiza esta tabla. Es mas confiable que
// GetAsyncKeyState desde otra goroutine (que no ve las teclas inyectadas por
// SendInput y puede quedar desfasada entre hilos).
var pressedKeys = struct {
	sync.Mutex
	m map[int32]bool
}{m: map[int32]bool{}}

func keyTracked(vk int32) bool {
	pressedKeys.Lock()
	defer pressedKeys.Unlock()
	return pressedKeys.m[vk]
}

func setKeyTracked(vk int32, down bool) {
	pressedKeys.Lock()
	if down {
		pressedKeys.m[vk] = true
	} else {
		delete(pressedKeys.m, vk)
	}
	pressedKeys.Unlock()
}

func clearTrackedKeys() {
	pressedKeys.Lock()
	pressedKeys.m = map[int32]bool{}
	pressedKeys.Unlock()
}

// logFile deja rastro en disco: sin ventana de terminal no hay otra forma de
// saber que esta haciendo la app cuando se lanza con doble click.
var logFile *os.File

func openLog() {
	if logFile != nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(filepath.Dir(exe), "mouse-teclado.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	logFile = f
}

func logf(format string, a ...any) {
	openLog()
	if logFile == nil {
		return
	}
	line := fmt.Sprintf("%s %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
	logFile.WriteString(line)
}

func debugf(format string, a ...any) {
	logf(format, a...)
	if st.verbose {
		log.Printf(format, a...)
	}
}

func mouseEvent(flags uint32, dx, dy int32, data uint32) input {
	return input{Type: 0, Mi: mouseInput{Dx: dx, Dy: dy, MouseData: data, DwFlags: flags}}
}

func buttonFlag(kind int, down bool) uint32 {
	switch kind {
	case btnLeft:
		if down {
			return fLeftDown
		}
		return fLeftUp
	case btnRight:
		if down {
			return fRightDown
		}
		return fRightUp
	default:
		if down {
			return fMiddleDown
		}
		return fMiddleUp
	}
}

func moveCursor(dx, dy int) { sendInput([]input{mouseEvent(fMove, int32(dx), int32(dy), 0)}) }

func pressButton(kind int) {
	sendInput([]input{mouseEvent(buttonFlag(kind, true), 0, 0, 0)})
}

func releaseButtonEvent(kind int) {
	sendInput([]input{mouseEvent(buttonFlag(kind, false), 0, 0, 0)})
}

func releaseAll() {
	if st.btnDown {
		releaseButtonEvent(st.btnKind)
	}
	st.btnDown = false
	st.btnKind = 0
}

func scroll(notches int, horizontal bool) {
	delta := uint32(notches * 120)
	flag := uint32(fWheel)
	if horizontal {
		flag = fHWheel
	}
	sendInput([]input{mouseEvent(flag, 0, 0, delta)})
}

// multiClick envia count down/up pairs: es lo que Windows interpreta como
// doble o triple click real.
func multiClick(kind, count int) {
	if count < 1 {
		count = 1
	}
	if count > 3 {
		count = 3
	}
	inputs := make([]input, 0, count*2)
	for i := 0; i < count; i++ {
		inputs = append(inputs,
			mouseEvent(buttonFlag(kind, true), 0, 0, 0),
			mouseEvent(buttonFlag(kind, false), 0, 0, 0))
	}
	sendInput(inputs)
}

// screenRect devuelve los limites de TODA la area de escritorio, incluyendo
// monitores secundarios.
func screenRect() (x, y, w, h int32) {
	x = systemMetrics(76) // SM_XVIRTUALSCREEN
	y = systemMetrics(77) // SM_YVIRTUALSCREEN
	w = systemMetrics(78) // SM_CXVIRTUALSCREEN
	h = systemMetrics(79) // SM_CYVIRTUALSCREEN
	if w <= 0 {
		w = systemMetrics(0)
	}
	if h <= 0 {
		h = systemMetrics(1)
	}
	return
}

// clampToScreen evita que el cursor se vaya de la pantalla y se pierda.
func clampToScreen() {
	sx, sy, sw, sh := screenRect()
	p, ok := cursorPos()
	if !ok {
		return
	}
	nx, ny := p.X, p.Y
	if nx < sx {
		nx = sx
	}
	if ny < sy {
		ny = sy
	}
	if nx > sx+sw-1 {
		nx = sx + sw - 1
	}
	if ny > sy+sh-1 {
		ny = sy + sh - 1
	}
	if nx != p.X || ny != p.Y {
		setCursorPos(nx, ny)
		debugf("clamp a %d,%d", nx, ny)
	}
}

// jumpToCorner lleva el cursor a un punto conocido de la pantalla.
func jumpToCorner(topLeft bool) {
	sx, sy, sw, sh := screenRect()
	x, y := sx+3, sy+3
	if !topLeft {
		x, y = sx+sw-4, sy+sh-4
	}
	setCursorPos(x, y)
	followCursor()
	debugf("salto a esquina")
}

func pressButtonAt(kind int) {
	p, _ := cursorPos()
	st.btnKind = kind
	st.btnDown = true
	st.btnDownTime = time.Now()
	st.btnDownPos = p
	pressButton(kind)
	debugf("down boton %d en %d,%d", kind, p.X, p.Y)
}

// releaseButtonAt decide click vs drag: un toque corto sin movimiento es click,
// cualquier otra cosa es arrastre.
func releaseButtonAt() {
	if !st.btnDown {
		return
	}
	kind := st.btnKind
	held := time.Since(st.btnDownTime) >= 220*time.Millisecond
	now, _ := cursorPos()
	dist := abs32(now.X-st.btnDownPos.X) + abs32(now.Y-st.btnDownPos.Y)
	moved := dist > 4

	releaseButtonEvent(kind)

	if kind == btnLeft && !held && !moved {
		count := 1
		if time.Since(st.lastClick) < 450*time.Millisecond {
			count = st.lastCount + 1
		}
		if count > 3 {
			count = 3
		}
		st.lastClick = time.Now()
		st.lastCount = count
		for i := 1; i < count; i++ {
			pressButton(btnLeft)
			releaseButtonEvent(btnLeft)
		}
		debugf("click x%d", count)
	} else if held || moved {
		debugf("fin de arrastre")
	}

	st.btnDown = false
	st.btnKind = 0
}

// adjustSpeed cambia la velocidad base con las teclas + y -.
func adjustSpeed(delta int) {
	st.speed += delta
	if st.speed < 1 {
		st.speed = 1
	}
	if st.speed > 200 {
		st.speed = 200
	}
	logf("velocidad base: %d px por paso", st.speed)
	updateTray()
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// tick se corre cada 15ms: mueve el cursor con las teclas presionadas, sin
// depender del auto-repeat del teclado.
func tick() {
	if st.cursorMode && st.autoExit > 0 && !st.lastActivity.IsZero() {
		if time.Since(st.lastActivity) > time.Duration(st.autoExit)*time.Second {
			debugf("auto-apagado por inactividad")
			setCursorMode(false)
			return
		}
	}
	// Watchdog: si un keyup se perdio (ventana elevada, cambio de foco), el
	// boton no puede quedar pegado presionado en Windows.
	if st.btnDown && !keyTracked(vkSpace) {
		logf("watchdog: el boton %d estaba presionado sin tecla, lo libero", st.btnKind)
		releaseButtonAt()
	}

	if !st.cursorMode {
		return
	}

	dx, dy := 0, 0
	if keyTracked(vkLeft) || keyTracked(vkKeyA) {
		dx--
	}
	if keyTracked(vkRight) || keyTracked(vkKeyD) {
		dx++
	}
	if keyTracked(vkUp) || keyTracked(vkKeyW) {
		dy--
	}
	if keyTracked(vkDown) || keyTracked(vkKeyS) {
		dy++
	}
	if dx == 0 && dy == 0 {
		// diagnostico: con -debug deja ver que ve el ticker cada 500 ms
		if st.verbose && time.Since(lastPollLog) > 500*time.Millisecond {
			lastPollLog = time.Now()
			logf("poll: cursorMode=%v right=%v left=%v down=%v shift=%v ctrl=%v",
				st.cursorMode, keyTracked(vkRight), keyTracked(vkLeft), keyTracked(vkDown),
				keyDown(vkShift), keyDown(vkControl))
		}
		return
	}

	// Velocidad con teclas dedicadas: F es turbo y Z es paso a 1 px.
	// Shift y Ctrl quedan libres para los atajos de la app de adelante, que es
	// lo que hace util tener teclado y mouse en el mismo aparato.
	fast := keyTracked(vkKeyF)
	precise := keyTracked(vkKeyZ)
	step := st.speed
	if fast {
		step = st.turbo
	}
	if precise {
		step = 1
	}

	if st.verbose {
		logf("mueve dx=%d dy=%d step=%d", dx*step, dy*step, step)
	}
	moveCursor(dx*step, dy*step)
	clampToScreen()
	followCursor()
}

func setCursorMode(on bool) {
	st.cursorMode = on
	if !on {
		releaseAll()
		hideIndicator()
	} else {
		st.lastActivity = time.Now()
		clearTrackedKeys()
		showIndicator()
	}
	updateTray()
	debugf("modo cursor: %v", on)
}

func toggleCursorMode() { setCursorMode(!st.cursorMode) }

func showHelp() {
	text := "Ctrl+Alt+M   cambiar teclado <-> mouse\n" +
		"Esc   salir del modo cursor\n" +
		"Ctrl+A/C/V/X/E/Z  seleccionar, copiar, pegar, editar\n" +
		"(las combinaciones con Ctrl, Shift o Alt funcionan)\n" +
		"Flechas / WASD  mover\n" +
		"F (pisado) = turbo    Z (pisado) = paso a 1 px\n" +
		"+ y -  cambian la velocidad base\n" +
		"Espacio  click izq.   Shift+Espacio  der.   Ctrl+Espacio  medio\n" +
		"Tocar Espacio 2-3 veces = doble / triple click\n" +
		"Mantener Espacio + mover = arrastrar (drag & drop)\n" +
		"RePag / AvPag  scroll    Alt+Arriba/Abajo  scroll\n" +
		"Alt+Inicio / Alt+Fin  scroll extremo\n" +
		"Alt+Shift+Izq/Der  scroll horizontal\n" +
		"Ctrl+Alt+Inicio / Fin  saltar a una esquina\n" +
		"Ctrl+Alt+H  esta ayuda    Ctrl+Alt+Q  salir\n\n" +
		"La cruceta roja marca siempre donde esta el cursor.\n" +
		"Fuera del modo cursor las teclas se comportan normal."
	messageBox(0, text, "Mouse por teclado", 0x40)
}

var _ = os.Getpid
var _ unsafe.Pointer
