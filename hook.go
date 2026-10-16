package main

import (
	"syscall"
	"unsafe"
)

// El hook de bajo nivel corre en el hilo que lo instalo: ese hilo necesita un
// bucle de mensajes, por eso main corre el loop en el hilo principal.
var hookHandle uintptr

// callbackProc se conserva vivo: si el delegate se recolecta, Windows deja de
// llamar al hook y se rompe en silencio.
var callbackProc uintptr

// installHook pone el hook global de teclado. Devuelve false si Windows lo
// rechaza (típico con apps elevadas si el proceso no es administrador).
func installHook() bool {
	callbackProc = syscall.NewCallback(hookTrampoline)
	h, ok := setWindowsHookEx(whKeyboardLL, callbackProc, moduleHandle(), 0)
	if !ok {
		return false
	}
	hookHandle = h
	logf("hook global de teclado instalado (hwnd=%d)", h)
	return true
}

func uninstallHook() {
	if hookHandle != 0 {
		pUnhookWindowsHookEx.Call(hookHandle)
		hookHandle = 0
	}
}

func hookTrampoline(nCode int32, wParam, lParam uintptr) uintptr {
	if nCode != hcAction {
		return callNextHookEx(hookHandle, nCode, wParam, lParam)
	}

	msg := int32(wParam)
	isDown := msg == wmKeyDown || msg == wmSysKeyDown
	isUp := msg == wmKeyUp || msg == wmSysKeyUp
	if !isDown && !isUp {
		return callNextHookEx(hookHandle, nCode, wParam, lParam)
	}

	k := (*kbdllhookstruct)(unsafe.Pointer(lParam)) //nolint:govet // KBDLLHOOKSTRUCT es la struct que pasa Windows
	vk := int32(k.VkCode)

	// estado real de teclas: el hook ve todas, incluidas las inyectadas
	setKeyTracked(vk, isDown)

	if st.verbose && isDown {
		logf("tecla vk=%d (0x%02X)", vk, vk)
	}

	ctrl := keyDown(vkControl)
	alt := keyDown(vkMenu)
	shift := keyDown(vkShift)

	// Atajos globales: funcionan con o sin modo cursor.
	if ctrl && alt && isDown {
		switch vk {
		case vkKeyM:
			toggleCursorMode()
			return 1
		case vkKeyQ:
			quitApp()
			return 1
		case vkKeyH:
			showHelp()
			return 1
		case vkKeyA, vkKeyD, vkKeyS, vkKeyW:
			// en modo cursor estas teclas son de movimiento: nunca escribir
			if st.cursorMode {
				return 1
			}
		}
	}

	// Escape siempre devuelve el teclado, este o no este el modo cursor.
	if vk == vkEscape && isDown && st.cursorMode {
		logf("Escape: salgo del modo cursor")
		setCursorMode(false)
		return 1
	}

	if !st.cursorMode {
		return callNextHookEx(hookHandle, nCode, wParam, lParam)
	}

	st.lastActivity = nowFunc()

	// Toggle configurable: tecla sola o combinacion con Ctrl/Alt/Shift.
	if vk == st.toggleVk && modMask() == st.toggleMod {
		if isDown {
			toggleCursorMode()
		}
		return 1
	}

	if vk == vkSpace {
		// Mantener Espacio dispara auto-repeat: llegan decenas de keydown y un
		// solo keyup. Si cada keydown mandara un "boton abajo" nuevo, el boton
		// queda pegado presionado en Windows y ningun click se registra.
		// Solo el primer keydown cuenta, y el keyup siempre cierra.
		if isDown {
			if k.Flags&0x00000001 != 0 {
				return 1 // repeticion del teclado: ignorar
			}
			if st.btnDown {
				return 1 // ya estaba presionado
			}
			// Boton que se manda:
			//   sin modificador      -> click izquierdo
			//   Ctrl  -> click derecho (menu contextual)
			//   Alt   -> click medio
			//   Shift -> click izquierdo con Shift, que es como se amplía
			//            una seleccion. No hace falta inyectar el modificador:
			//            el usuario lo tiene apretado de verdad y Windows lo lee
			//            del estado real cuando llega el click.
			kind := btnLeft
			if ctrl {
				kind = btnRight
			} else if alt {
				kind = btnMiddle
			}
			logf("espacio abajo: boton %d en %d,%d (modo cursor %v)",
				kind, cursorX(), cursorY(), st.cursorMode)
			pressButtonAt(kind)
		} else {
			logf("espacio arriba (boton presionado=%v)", st.btnDown)
			releaseButtonAt()
			followCursor()
		}
		return 1
	}

	if isDown {
		repeat := k.Flags&1 != 0

		// Velocidad: + y - cambian la velocidad base, F turbo y Z precision.
		// Solo cuando la tecla va sola: con Ctrl/Shift/Alt encima pasa de largo
		// para no romper atajos como Ctrl+= (zoom) o Ctrl+- .
		if !ctrl && !shift && !alt {
			switch vk {
			case vkOemPlus:
				if !repeat {
					adjustSpeed(speedStep)
				}
				return 1
			case vkOemMinus:
				if !repeat {
					adjustSpeed(-speedStep)
				}
				return 1
			case vkKeyF, vkKeyZ:
				return 1 // solo existen como modificadores de velocidad
			case vkTab:
				toggleArrowMode()
				return 1
			}
		}

		// volver a un punto conocido si el cursor se perdio
		if ctrl && alt && (vk == vkHome || vk == vkEnd) {
			if !repeat {
				jumpToCorner(vk == vkHome)
			}
			return 1
		}

		if vk == vkPrior {
			if !repeat {
				scroll(3, false)
			}
			return 1
		}
		if vk == vkNext {
			if !repeat {
				scroll(-3, false)
			}
			return 1
		}

		if alt {
			switch vk {
			case vkUp:
				if !repeat {
					scroll(3, false)
				}
				return 1
			case vkDown:
				if !repeat {
					scroll(-3, false)
				}
				return 1
			case vkLeft:
				if shift && !repeat {
					scroll(-3, true)
				}
				return 1
			case vkRight:
				if shift && !repeat {
					scroll(3, true)
				}
				return 1
			case vkHome:
				if !repeat {
					scroll(30, false)
				}
				return 1
			case vkEnd:
				if !repeat {
					scroll(-30, false)
				}
				return 1
			}
		}

		// en modo cursor el teclado deja de escribir estas teclas
		switch vk {
		case vkLeft, vkRight, vkUp, vkDown,
			vkKeyA, vkKeyD, vkKeyS, vkKeyW,
			vkHome, vkEnd:
			return 1
		}
	}

	// Modo mouse estricto: en modo cursor solo pasan las teclas del mouse.
	// Todo lo demas se traga, asi no se activen atajos de la app de
	// adelante ni se escriba nada por accidente.
	if st.cursorMode && !st.freeKeys {
		// los modificadores siempre pasan: si no, Windows cree que quedaron
		// pegados y rompe cualquier combinacion
		if isModifierKey(vk) {
			return callNextHookEx(hookHandle, nCode, wParam, lParam)
		}
		if isMouseModeKey(vk) {
			return callNextHookEx(hookHandle, nCode, wParam, lParam)
		}
		// Cualquier combinacion con Ctrl, Shift o Alt pasa: asi se puede
		// seleccionar con el mouse y despues copiar o pegar (Ctrl+A, Ctrl+C,
		// Ctrl+V, Ctrl+X, Ctrl+E, Ctrl+Z...) como si el teclado y el mouse
		// estuvieran juntos. Lo que se traga sigue siendo la tecla suelta, que
		// es la que se escapa por accidente.
		if ctrl || shift || alt {
			return callNextHookEx(hookHandle, nCode, wParam, lParam)
		}
		if isDown {
			logf("tecla vk=%d tragada por el modo mouse", vk)
		}
		return 1 // me trago el keydown y su keyup
	}

	return callNextHookEx(hookHandle, nCode, wParam, lParam)
}

// isModifierKey: los modificadores tienen que pasar siempre.
func isModifierKey(vk int32) bool {
	switch vk {
	case vkShift, vkControl, vkMenu, 0x5B, 0x5C, // LWin, RWin
		0xA0, 0xA1, 0xA2, 0xA3, 0xA4, 0xA5: // L/R shift, ctrl, alt
		return true
	}
	return false
}

// speedStep es cuanto sube o baja la velocidad con cada pulsacion de + o -.
const speedStep = 3

// isMouseModeKey: en modo mouse solo estas teclas hacen algo.
func isMouseModeKey(vk int32) bool {
	switch vk {
	case vkSpace, // click
		vkLeft, vkRight, vkUp, vkDown, // mover
		vkKeyA, vkKeyD, vkKeyS, vkKeyW, // mover
		vkPrior, vkNext, // scroll
		vkHome, vkEnd, // scroll / esquinas
		vkEscape, // salir del modo mouse
		0x2C:     // PrintScreen
		return true
	}
	return false
}
