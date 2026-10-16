# Mouse por teclado

[English](README.md) | **Espanol**

Cursor virtual **de toda la pantalla de Windows**, controlado 100% por teclado.
Sin ventana de terminal: es un `.exe` nativo que vive en la bandeja del sistema.

Un solo ejecutable, sin dependencias externas (solo la librería estándar de Go).

---

## Instalación

```powershell
mouse-teclado.exe -install     # arranca con Windows
mouse-teclado.exe -uninstall   # sacarlo
```

`-install` deja un hard link del `.exe` en la carpeta Inicio de Windows. Windows
ejecuta todo lo que encuentra ahí, y como el hard link apunta al mismo archivo,
una versión nueva del `.exe` no deja el acceso viejo desactualizado.

Para arrancarlo ahora mismo: doble click en el `.exe`, o `Win+R` y la ruta
completa.

Verificación sin tocar el mouse (con la app cerrada):

```powershell
mouse-teclado.exe -selftest
```

---

## Cómo se usa

### Cambiar de teclado a modo mouse

`Ctrl+Alt+M` alterna entre dos modos:

- **Modo teclado** — todo se comporta normal. Escribís con `W`, `A`, `S`, `D` y
  `Espacio` sin que pase nada raro.
- **Modo cursor** — el puntero se mueve con las teclas. Las teclas sueltas se
  tragan, así que no se escribe nada ni se activa nada por accidente en la app de
  adelante, pero **toda combinación con `Ctrl`, `Shift` o `Alt` sigue
  funcionando**: seleccionás texto con el mouse y después `Ctrl+A`, `Ctrl+C`,
  `Ctrl+V`, `Ctrl+X`, `Ctrl+E`, `Ctrl+Z`, como si teclado y mouse fueran el mismo
  aparato.

Tres formas de volver al modo teclado:

| Atajo | Efecto |
| --- | --- |
| `Ctrl+Alt+M` | Cambiar teclado ↔ mouse |
| `Esc` | Salir del modo cursor |
| — | Se apaga solo a los 45 s sin uso |

`Ctrl+Alt+Q` cierra la app entera.

También se puede desde la bandeja: click izquierdo sobre el icono alterna el modo,
click derecho abre el menú (activar/desactivar, ayuda, salir).

### Teclas del modo cursor

| Tecla | Acción |
| --- | --- |
| Flechas, `W` `A` `S` `D` | Mover el puntero |
| `Shift` + mover | Turbo (rápido) |
| `Ctrl` + mover | Paso a 1 px, para apuntar justo |
| `Espacio` | Click primario |
| `Shift` + `Espacio` | Click secundario (menú contextual) |
| `Ctrl` + `Espacio` | Click medio |
| `Espacio` 2-3 veces seguidas | Doble / triple click |
| Mantener `Espacio` + mover | Arrastrar (drag & drop) |
| `RePág` / `AvPág` | Scrollear |
| `Alt` + `↑` / `↓` | Scrollear |
| `Alt` + `Inicio` / `Fin` | Scroll al extremo |
| `Alt` + `Shift` + `←` / `→` | Scroll horizontal |
| `Ctrl+Alt+Inicio` / `Ctrl+Alt+Fin` | Saltar a una esquina de la pantalla |
| `Ctrl`+`A`/`C`/`V`/`X`/`E`/`Z` y cualquier otra combinación | Seleccionar, copiar, pegar, cortar, editar, deshacer |
| `Ctrl+Alt+H` | Ayuda en pantalla |
| `Ctrl+Alt+Q` | Cerrar la app |

Atajos nativos de Windows que ya cubren el resto sin mouse: `Win`+flechas
(mover/encoger ventana), `Alt+F7` (mover ventana con las flechas), `Alt+Tab`,
`Win+D`, `Ctrl+Shift+←/→` (seleccionar texto).

---

## Decisiones de diseño

### Por qué los clicks son así

El pedido original era `Ctrl+Espacio` para click primario y `Shift+Espacio` para
el secundario. La versión final mejora dos cosas:

- **Un solo modo** (`Ctrl+Alt+M`) para mover el mouse con el teclado. Fuera del
  modo no se intercepta nada: `W`, `A`, `S`, `D` y `Espacio` escriben normal.
- **`Espacio` es el click primario**, porque es la tecla natural bajo el pulgar.
  El conflicto entre click y arrastre se resuelve por **duración**, no por
  combinaciones: tocar es click, mantener y mover es drag.
- **Click secundario y medio** quedan en `Shift+Espacio` y `Ctrl+Espacio`, tal
  como se pidió, pero solo dentro del modo cursor para no romper combinaciones
  que usan editores y terminales.

### Por qué el modo cursor traga las teclas sueltas pero deja los atajos

Pedido explícito: en modo mouse no debe funcionar ninguna otra tecla, "para que
no haya errores". Así, un `C` suelto no borra una palabra ni dispara un atajo de
la app de adelante.

Lo que se bloquea es la **tecla suelta**. Cualquier combinación que lleve `Ctrl`,
`Shift` o `Alt` pasa de largo, porque eso es lo que hace la herramienta usable con
las dos manos en un solo aparato: seleccionás texto con el mouse — arrastrando, o
click antes y `Shift`+click después — y después apretás `Ctrl+C` para copiar o
`Ctrl+V` para pegar. Los modificadores sueltos también pasan siempre: si se
tragaran, Windows creería que quedaron pegados y rompería cualquier combinación.

Si preferís el comportamiento anterior, arrancala con `-freekeys`.

### Por qué una cruceta roja

Un mouse relativo se puede ir de la pantalla y desaparecer. La cruceta marca
siempre dónde está el puntero, y hay dos medidas más:

- **Límite de pantalla**: el cursor nunca sale de los bordes.
- **Saltos a esquina** con `Ctrl+Alt+Inicio` / `Ctrl+Alt+Fin`, para volver a un
  punto conocido en un toque.

La cruceta es una ventana *layered* con alfa por píxel (ver
[historial de bugs](#historial-de-bugs-la-raíz-de-cada-problema)) para que no se
coma los clicks.

---

## Opciones

| Flag | Por defecto | Qué hace |
| --- | --- | --- |
| `-speed` | `12` | píxeles por paso |
| `-turbo` | `60` | píxeles por paso con `Shift` |
| `-autoexit` | `45` | segundos de inactividad para apagar el modo cursor (`0` = nunca) |
| `-togglekey` | `Ctrl+Alt+M` | tecla o combinación para cambiar de modo |
| `-nocross` | off | no muestra la cruceta roja |
| `-freekeys` | off | en modo cursor deja pasar todas las teclas |
| `-debug` | off | log detallado (cada tecla y cada movimiento) |
| `-version` | — | muestra la versión y sale |
| `-selftest` | — | verifica hook y `SendInput` sin mover el cursor |
| `-install` / `-uninstall` | — | arranque automático |

`-togglekey` acepta una tecla sola (`F1`-`F24`, `Pause`, `ScrollLock`,
`CapsLock`, `Insert`, `Home`, `End`, `PageUp`, `PageDown`, `Space`, las flechas o
una letra) o una combinación (`Ctrl+Alt+M`, `Shift+Space`). Como depende del
teclado que tengas, se cambia sin tocar código:

```powershell
mouse-teclado.exe -togglekey F13
```

---

## Diagnóstico

La app escribe `mouse-teclado.log` junto al `.exe`. Es la única forma de ver qué
pasa cuando no hay terminal:

```
22:02:32 inicio: pid=7976 toggle=Ctrl+Alt+M speed=12 turbo=60 autoexit=45
22:02:32 hilo principal fijado con LockOSThread (OS thread=3104)
22:02:32 hook global de teclado instalado (hwnd=7537639)
22:02:32 bandeja creada: icono en el sistema, mensaje de bienvenida enviado
```

Con `-debug` además loguea cada tecla (`tecla vk=32`), cada movimiento
(`mueve dx=12 dy=0 step=12`) y qué teclas ve el ticker cada 500 ms.

**Después de recompilar, volvé a correr `mouse-teclado.exe -install`**: el hard
link de la carpeta Inicio sigue apuntando al `.exe` viejo si el binario fue
reemplazado.

### Herramientas de prueba

En `%TEMP%` quedan tres programas usados para verificar la app sin mouse:

| Carpeta | Qué hace |
| --- | --- |
| `clicksink` | Ventana que registra los clicks que recibe; sirve para comprobar que el click virtual llega a una app real |
| `sendkey2` | Inyecta teclas reales (`SendInput`) y consulta el estado de la app por el log |
| `modestrict` | Verifica que el modo mouse traga las teclas que no son del mouse |

Dos cosas que conviene saber si los usás:

- La ventana de prueba tiene que estar **arriba**: si queda tapada por otra, no
  recibe ni los `WM_MOUSEMOVE` y el diagnóstico da falsos negativos.
- **`SendInput` con `VK_ESCAPE` no lo recibe el hook de bajo nivel**: devuelve 1
  (evento entregado) pero ningún hook ve `vk=27`, ni siquiera mandando el código
  de barrido explícito. Hay teclas que no se pueden verificar por inyección.

---

## Historial de bugs: la raíz de cada problema

Estos cuatro problemas costaron encontrar. Están documentados porque el código
parece raro sin el porqué, y volver a tropezar con ellos sería una pérdida de
tiempo.

### 1. El hook nunca disparaba

**Síntoma:** la app arrancaba, el log decía "hook instalado", pero ninguna tecla
llegaba.

**Causa:** un hook de bajo nivel se registra en un hilo de Windows y recibe sus
eventos **en ese mismo hilo**. Go mueve los goroutines entre hilos cuando quiere,
así que el bucle de mensajes podía terminar bombeando una cola que no era la del
hook.

**Arreglo:** `runtime.LockOSThread()` al arrancar el hilo principal.

### 2. Las flechas sí, las letras no

**Síntoma:** con el hook ya funcionando, las flechas movían el puntero pero
`W A S D` no.

**Causa:** el estado de las teclas se consultaba con `GetAsyncKeyState` desde la
goroutine del ticker. Ese estado asíncrono no ve las teclas inyectadas y puede
quedar desfasado entre hilos.

**Arreglo:** el propio hook llena `pressedKeys` con cada keydown/keyup que llega,
y el ticker lee de ahí. Los modificadores (`Shift`, `Ctrl`) siguen viniendo de
`GetAsyncKeyState` a propósito: son los que más expuestos quedan a quedar
"pegados" si algún keyup no llega.

### 3. El click no se registraba y el botón quedaba pegado

**Síntoma:** el puntero se movía pero ningún click llegaba a las apps.

**Causa:** mantener `Espacio` dispara el auto-repeat del teclado: decenas de
`WM_KEYDOWN` con el flag de repetición y un solo `WM_KEYUP`. Cada keydown mandaba
un "botón abajo" nuevo, así que el botón izquierdo quedaba **pegado presionado**
en Windows y todo click posterior se perdía.

**Arreglo:** solo cuenta el primer keydown (`Flags & 1`), se ignora el keydown si
el botón ya está presionado, y hay un watchdog que libera el botón si la tecla
dejó de estar.

### 4. La cruceta se comía todos los clicks

**Síntoma:** con la cruceta activa, ningún click llegaba a la app de abajo. Sin
cruceta (`-nocross`), todo funcionaba.

**Causa:** la cruceta era una ventana normal (`WS_POPUP`) encima del puntero.
Cualquier ventana así se queda con los clicks: el movimiento es global, pero el
click va a la ventana que está arriba.

**Arreglo:** ventana *layered* dibujada en un DIB 32bpp con alfa por píxel —
solo los píxeles de la cruz tienen alfa 255, el resto es transparente y Windows
deja pasar el click. `WS_EX_TRANSPARENT` y `WM_NCHITTEST`/`HTTRANSPARENT` van
como refuerzo, no como solución.

Extra que costó: `UpdateLayeredWindow` **no deja la ventana visible por sí solo**,
hay que llamar `ShowWindow` después, o la cruceta existe pero no se ve.

---

## Límites conocidos

- **Ventanas elevadas:** si la app de adelante corre como administrador, Windows
  bloquea los clicks hacia ella (protección UIPI). El puntero se mueve igual,
  pero el click no se registra. Se resuelve no corriendo esas apps elevadas.
- **Pantalla segura de UAC:** es otra sesión de escritorio, no es controlable.
- **Precisión:** el movimiento es por teclas presionadas cada 15 ms. Se siente
  fluido, pero para apuntar con exactitud hay que usar `Ctrl` (pasos de 1 px).
- **Arranque automático por carpeta Inicio, no por registro:** en esta máquina
  las escrituras al registro hechas por procesos hijos se descartan en silencio e
  incluso llegan a romper el proceso. La carpeta Inicio funciona siempre.

---

## Compilar

```powershell
go build -ldflags "-H windowsgui -s -w" -o mouse-teclado.exe .
```

`-H windowsgui` es lo que evita la ventana de consola: sin eso el `.exe` abre una
terminal.

Después de compilar, recordá reinstalar el arranque automático
(`mouse-teclado.exe -install`).

---

## Mapa del código

| Archivo | Responsabilidad |
| --- | --- |
| `main.go` | Flags, arranque, instancia única, autoverificación, instalación |
| `hook.go` | Hook global de teclado (`WH_KEYBOARD_LL`): qué tecla hace qué |
| `mouse.go` | Movimiento, clicks, scroll, límites de pantalla, estado de teclas |
| `indicator.go` | Cruceta roja click-through (ventana layered + DIB) |
| `tray.go` | Bandeja del sistema y menú |
| `win32.go` | Declaraciones de la API de Windows (structs, procs, constantes) |

Todo el movimiento pasa por un ticker de 15 ms (`ticker` en `main.go`), que lee
el estado de teclas que armó el hook.