package main

// Mouse por teclado: cursor virtual global de Windows controlado por teclado.
//
//	Ctrl+Alt+M   cambiar teclado <-> modo cursor
//	 Flechas / WASD  mover      Shift = turbo      Ctrl = 1 px
//	 Espacio = click izq.  Shift+Espacio = der.  Ctrl+Espacio = medio
//	 mantener Espacio + mover = arrastrar
//
// La app vive en la bandeja del sistema: sin ventanas de terminal.

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	nowFunc = time.Now

	optInstall   = flag.Bool("install", false, "registra el arranque automatico de Windows y sale")
	optUninstall = flag.Bool("uninstall", false, "saca el arranque automatico y sale")
	optSelfTest  = flag.Bool("selftest", false, "verifica hook y SendInput sin mover el cursor")
	optSpeed     = flag.Int("speed", 12, "pixeles por paso")
	optTurbo     = flag.Int("turbo", 60, "pixeles por paso con Shift")
	optAutoExit  = flag.Int("autoexit", 45, "segundos de inactividad para apagar el modo cursor (0 = nunca)")
	optToggleKey = flag.String("togglekey", "Ctrl+Alt+M", "tecla o combinacion para cambiar teclado <-> mouse")
	optNoCross   = flag.Bool("nocross", false, "no muestra la cruceta roja del cursor")
	optFreeKeys  = flag.Bool("freekeys", false, "en modo cursor deja pasar todas las teclas (por defecto solo las del mouse)")
	optDebug     = flag.Bool("debug", false, "log detallado en stderr")
	optVersion   = flag.Bool("version", false, "muestra la version y sale")
)

const (
	version            = "1.0.0"
	runKeyPath         = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName       = "MouseTeclado"
	singleMutex        = "Local\\MouseTecladoUnico"
	errorAlreadyExists = 183
)

func main() {
	flag.Parse()

	if *optVersion {
		fmt.Printf("mouse-teclado %s\n", version)
		return
	}

	if *optInstall {
		if err := installAutostart(); err != nil {
			fmt.Println("error al instalar el arranque automatico:", err)
			os.Exit(1)
		}
		target, _ := startupLinkPath()
		fmt.Println("Arranque automatico instalado:", target)
		return
	}

	if *optUninstall {
		if err := uninstallAutostart(); err != nil {
			fmt.Println("error al quitar el arranque automatico:", err)
			os.Exit(1)
		}
		fmt.Println("Arranque automatico eliminado")
		return
	}

	vk, mod, err := resolveToggleSpec(*optToggleKey)
	if err != nil {
		messageBox(0, err.Error(), "Mouse por teclado", 0x10)
		fmt.Println(err)
		os.Exit(1)
	}

	st.speed = max(1, *optSpeed)
	st.turbo = max(st.speed, *optTurbo)
	st.autoExit = *optAutoExit
	st.toggleVk = vk
	st.toggleMod = mod
	st.showCross = !*optNoCross
	st.freeKeys = *optFreeKeys
	st.verbose = *optDebug

	mutex, err := acquireSingleInstance()
	if err != nil {
		// Ya hay una instancia. Con doble click no hay salida de consola, asi
		// que sin este aviso el doble click parece no hacer nada.
		logf("instancia unica: ya hay una corriendo, no se inicia otra")
		if *optSelfTest {
			fmt.Println("ya hay una instancia corriendo; el selftest necesita la app cerrada")
			os.Exit(2)
		}
		messageBox(0,
			"Mouse por teclado ya esta corriendo.\n\n"+
				"Buscalo en la bandeja del sistema (abajo a la derecha, puede estar "+
				"detras de la flecha ^).\n\n"+
				"Click izquierdo: activar o desactivar el modo cursor.",
			"Mouse por teclado", 0x40)
		return
	}
	logf("inicio: pid=%d toggle=%s speed=%d turbo=%d autoexit=%d",
		os.Getpid(), *optToggleKey, st.speed, st.turbo, st.autoExit)
	defer releaseMutex(mutex)

	if *optSelfTest {
		os.Exit(runSelfTest())
	}

	// CRITICO: un hook de bajo nivel se registra en el hilo de Windows que lo
	// instala y recibe sus eventos en ese mismo hilo. Go puede mover el
	// goroutine a otro hilo en cualquier preempcion; entonces el bucle de
	// mensajes bombearia una cola que no es la del hook, el hook no volveria a
	// disparar y el mouse virtual pareceria muerto. LockOSThread lo evita.
	runtime.LockOSThread()
	logf("hilo principal fijado con LockOSThread (OS thread=%d)", getCurrentThreadId())

	if !installHook() {
		logf("ERROR: el hook global de teclado no se pudo instalar")
		messageBox(0,
			"No se pudo instalar el hook global de teclado.\n\n"+
				"Ceria las ventanas abiertas como administrador y volve a intentar.",
			"Mouse por teclado", 0x10)
		os.Exit(1)
	}
	defer uninstallHook()

	showTray()
	defer removeTray()
	if !trayPresent {
		logf("ERROR: no se pudo crear el icono en la bandeja")
		messageBox(0,
			"No se pudo crear el icono en la bandeja del sistema.\n\n"+
				"Los atajos siguen funcionando: Ctrl+Alt+M (activar/desactivar modo cursor), "+
				"Ctrl+Alt+H (ayuda) y Ctrl+Alt+Q (salir).",
			"Mouse por teclado", 0x30)
	} else {
		logf("bandeja creada: icono en el sistema, mensaje de bienvenida enviado")
		showWelcomeBalloon()
	}

	// El movimiento corre en su propia goroutine: el hook necesita el hilo
	// principalny el bucle de mensajes.
	stop := make(chan struct{})
	go ticker(stop)

	runMessageLoop()

	close(stop)
	releaseAll()
	hideIndicator()
	removeTray()
	uninstallHook()
}

func ticker(stop <-chan struct{}) {
	// 15 ms entre pasos: se siente fluido sin gastar CPU.
	const interval = 15 * time.Millisecond
	for {
		select {
		case <-stop:
			return
		case <-time.After(interval):
			tick()
		}
	}
}

func runMessageLoop() {
	var m msg
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// quitApp se puede llamar desde el menu de la bandeja o desde el hook.
func quitApp() {
	removeTray()
	releaseAll()
	hideIndicator()
	pPostQuitMessage.Call(0)
}

// ---------- instancia unica

func acquireSingleInstance() (syscall.Handle, error) {
	name, _ := syscall.UTF16PtrFromString(singleMutex)
	h, _, err := pCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))
	handle := syscall.Handle(h)
	if handle == 0 {
		return handle, err
	}
	// El error de Win32 lo devuelve el propio Call, capturado justo despues de
	// la llamada; consultar GetLastError despues puede leer otra cosa.
	if err == syscall.Errno(errorAlreadyExists) {
		pCloseHandle.Call(h)
		return handle, syscall.EWINDOWS
	}
	return handle, nil
}

func releaseMutex(h syscall.Handle) {
	pCloseHandle.Call(uintptr(h))
}

// ---------- autoverificacion

func runSelfTest() int {
	if !installHook() {
		fmt.Println("SELFTEST: FAIL - el hook global no se pudo instalar")
		return 1
	}
	defer uninstallHook()

	size := int(unsafe.Sizeof(input{}))
	// un evento de rueda: prueba que SendInput llega al sistema
	before, _ := cursorPos()
	wheel := sendInput([]input{mouseEvent(fWheel, 0, 0, 120)})
	time.Sleep(1200 * time.Millisecond)
	after, _ := cursorPos()

	moved := before.X != after.X || before.Y != after.Y
	fmt.Printf("INPUT struct: %d bytes (x64 esperado 40)\n", size)
	fmt.Printf("SendInput rueda: %d evento(s)\n", wheel)
	fmt.Printf("Cursor antes: %d,%d  despues: %d,%d\n", before.X, before.Y, after.X, after.Y)

	if size != 40 || wheel != 1 || moved {
		fmt.Println("SELFTEST: FAIL")
		return 1
	}
	fmt.Println("SELFTEST: OK (hook instalado, SendInput responde, el cursor no deriva)")
	return 0
}

// ---------- arranque automatico (carpeta Inicio)
//
// Windows ejecuta todo lo que encuentra en la carpeta Inicio del usuario, asi
// que ahi va un hard link del ejecutable. Se eligio esta via y no la clave
// HKCU Run porque en esta maquina las escrituras al registro hechas desde
// procesos hijos se descartan en silencio e incluso llegan a romper el proceso.

func startupLinkPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Microsoft", "Windows", "Start Menu", "Programs",
		"Startup", "MouseTeclado.exe"), nil
}

// installAutostart deja el ejecutable enlazado en la carpeta Inicio. El hard
// link apunta al mismo archivo, asi que una version nueva del .exe no deja el
// acceso viejo desactualizado.
func installAutostart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	target, err := startupLinkPath()
	if err != nil {
		return err
	}
	_ = os.Remove(target)

	if err := os.Link(exe, target); err == nil {
		return nil
	}
	// algunos volumenes no admiten hard links: cae a una copia
	data, readErr := os.ReadFile(exe)
	if readErr != nil {
		return fmt.Errorf("no se pudo enlazar en Inicio: %v", err)
	}
	if writeErr := os.WriteFile(target, data, 0o755); writeErr != nil {
		return fmt.Errorf("no se pudo escribir en Inicio: %w", writeErr)
	}
	return nil
}

func uninstallAutostart() error {
	target, err := startupLinkPath()
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ---------- resolucion de la tecla de toggle

var modifierBits = map[string]int{
	"ctrl": 1, "control": 1, "alt": 2, "menu": 2, "shift": 4,
}

var namedKeys = map[string]int32{
	"pause": vkPause, "break": vkPause, "space": vkSpace, "tab": 0x09, "enter": 0x0D,
	"insert": 0x2D, "delete": 0x2E, "home": vkHome, "end": vkEnd,
	"pageup": vkPrior, "pagedown": vkNext, "up": vkUp, "down": vkDown,
	"left": vkLeft, "right": vkRight, "scrolllock": 0x91, "printscreen": 0x2C,
	"apps": 0x5D, "sleep": 0x5F,
}

// resolveToggleSpec convierte "Ctrl+Alt+M", "F13" o "Shift+Space" en el virtual
// key y la mascara de modificadores (1=Ctrl, 2=Alt, 4=Shift).
func resolveToggleSpec(spec string) (int32, int, error) {
	parts := strings.Split(spec, ",")
	chunk := strings.TrimSpace(parts[len(parts)-1])
	keys := strings.Split(chunk, "+")
	for i := range keys {
		keys[i] = strings.TrimSpace(keys[i])
	}
	last := keys[len(keys)-1]

	vk, err := resolveKeyToVK(last)
	if err != nil {
		return 0, 0, err
	}

	mask := 0
	for _, m := range keys[:len(keys)-1] {
		bit, ok := modifierBits[strings.ToLower(m)]
		if !ok {
			return 0, 0, fmt.Errorf("modificador no soportado: %q. Use Ctrl, Alt o Shift", m)
		}
		mask |= bit
	}
	return vk, mask, nil
}

func resolveKeyToVK(name string) (int32, error) {
	k := strings.ToLower(strings.TrimSpace(name))
	if v, ok := namedKeys[k]; ok {
		return v, nil
	}
	low := strings.TrimPrefix(k, "f")
	if k != low && len(low) >= 1 && len(low) <= 2 {
		n := 0
		for _, c := range low {
			if c < '0' || c > '9' {
				n = -1
				break
			}
			n = n*10 + int(c-'0')
		}
		if n >= 1 && n <= 24 {
			return int32(0x6F + n), nil
		}
	}
	if len(k) == 1 && k[0] >= 'a' && k[0] <= 'z' {
		return int32(strings.ToUpper(k)[0]), nil
	}
	return 0, fmt.Errorf("tecla no soportada para toggle: %q. Use F1-F24, Pause, ScrollLock, "+
		"Insert, Home, End, PageUp, PageDown, Space, las flechas o una letra", name)
}
