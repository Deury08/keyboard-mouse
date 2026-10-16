# Keyboard Mouse

A virtual cursor for the **whole Windows screen**, driven 100% by keyboard.
No terminal window: it is a native `.exe` that lives in the system tray.

A single executable, no external dependencies (Go standard library only).

**[English](README.md) | [Español](README.es.md)**

---

## Install

```powershell
mouse-teclado.exe -install     # start with Windows
mouse-teclado.exe -uninstall   # remove it
```

`-install` drops a hard link of the `.exe` into your Windows Startup folder.
Windows runs whatever it finds there, and because a hard link points to the same
file, a new build never leaves a stale shortcut behind.

To start it right now: double click the `.exe`, or `Win+R` and paste the full
path.

Check that it works without touching a mouse (with the app closed):

```powershell
mouse-teclado.exe -selftest
```

---

## Usage

### Switching between keyboard and mouse mode

`Ctrl+Alt+M` toggles between two modes:

- **Keyboard mode** — everything behaves normally. You type with `W`, `A`, `S`,
  `D` and `Space` and nothing weird happens.
- **Cursor mode** — the pointer moves with the keyboard. Bare keys are swallowed,
  so nothing gets typed and nothing gets triggered by accident in whatever app is
  in front, but **every combination with `Ctrl`, `Shift` or `Alt` still works**:
  select text with the mouse and then `Ctrl+A`, `Ctrl+C`, `Ctrl+V`, `Ctrl+X`,
  `Ctrl+E`, `Ctrl+Z`, as if keyboard and mouse were the same device.

Three ways back to keyboard mode:

| Shortcut | Effect |
| --- | --- |
| `Ctrl+Alt+M` | Toggle keyboard ↔ mouse |
| `Esc` | Leave cursor mode |
| — | Turns itself off after 45 s of inactivity |

`Ctrl+Alt+Q` quits the app entirely.

You can also use the tray icon: left click toggles the mode, right click opens
the menu (toggle, help, quit).

### Cursor mode keys

| Key | Action |
| --- | --- |
| Arrow keys, `W` `A` `S` `D` | Move the pointer (or scroll and zoom — see `Tab`) |
| `Tab` | Switch what the arrow keys do: pointer ↔ scroll + zoom |
| Hold `F` while moving | Turbo (fast) |
| Hold `Z` while moving | 1 px steps, for precise aiming |
| `+` / `-` | Make the base speed faster or slower |
| `Space` | Primary click |
| `Shift` + `Space` | Primary click **with Shift**: extend the selection |
| `Ctrl` + `Space` | Secondary click (context menu) |
| `Alt` + `Space` | Middle click |
| `Space` 2-3 times | Double / triple click |
| Hold `Space` + move | Drag and drop |
| `PgUp` / `PgDn` | Scroll (always available, whatever `Tab` is set to) |
| `Alt` + `↑` / `↓` | Scroll |
| `Alt` + `Home` / `End` | Scroll to the extreme |
| `Alt` + `Shift` + `←` / `→` | Horizontal scroll |
| `Ctrl+Alt+Home` / `Ctrl+Alt+End` | Jump to a screen corner |
| `Ctrl`+`A`/`C`/`V`/`X`/`E`/`Z` and any other combination | Select, copy, paste, cut, edit, undo |
| `Ctrl+Alt+H` | On-screen help |
| `Ctrl+Alt+Q` | Quit |

Native Windows shortcuts that already cover the rest without a mouse: `Win`+arrows
(move/resize window), `Alt+F7` (move window with arrows), `Alt+Tab`, `Win+D`,
`Ctrl+Shift+←/→` (select text).

---

## Design decisions

### Why the clicks work this way

The original request was `Ctrl+Space` for the primary click and `Shift+Space` for
the secondary one. The final version improves two things:

- **One single mode** (`Ctrl+Alt+M`) to move the mouse with the keyboard. Outside
  that mode nothing is intercepted: `W`, `A`, `S`, `D` and `Space` type normally.
- **`Space` is the primary click**, because it is the natural key under your
  thumb. The click-versus-drag conflict is resolved by **duration**, not by key
  combinations: a tap is a click, holding and moving is a drag.
- **Secondary and middle click** stay on `Shift+Space` and `Ctrl+Space`, exactly
  as requested, but only inside cursor mode so they do not steal combinations that
  editors and terminals rely on.

### The on-screen legend

Every time cursor mode starts, a small legend appears at the top of the screen
with the current key map, and disappears on its own after 9 seconds. It is a
layered window with a transparent margin and `HTTRANSPARENT`, so it never steals
a click, and it hides as soon as you leave the mode. Start the app with
`-nolegend` if you would rather not see it.

For the full help at any moment: `Ctrl+Alt+H`.

### Why `Tab` switches the meaning of the arrow keys

Arrows can only mean one thing at a time, and the three things actually needed
are: move the pointer, scroll the content, and zoom. `Tab` alternates between
the first one and the other two:

| Mode | `↑` `↓` | `←` `→` | `W` `S` / `A` `D` |
| --- | --- | --- | --- |
| **POINTER** (default) | move | move | same as the arrows |
| **SCROLL + ZOOM** | scroll | zoom | same as the arrows |

Press `Tab` again to go back. It starts in pointer mode every time cursor mode is
activated, the tray tooltip says which mode is active, and the on-screen legend
comes back showing the change.

Zoom is injected as `Ctrl`+wheel, which is what browsers, Office and most PDF
readers understand. Windows has no universal zoom keystroke, so in apps that do
not use that combination the arrows will simply do nothing in that mode.

### Why speed lives on its own keys

`Shift` and `Ctrl` started out as the speed modifiers — `Shift` for turbo, `Ctrl`
for pixel-perfect steps — but those are exactly the keys that carry `Shift+click`
selection and `Ctrl+C` copy. Using them for both made them ambiguous.

So speed moved to its own keys: hold `F` for turbo, hold `Z` for 1 px steps, and
`+` / `-` to tune the base speed (also visible in the tray tooltip). `Shift` and
`Ctrl` are now free for whatever the app in front expects.

`+` and `-` are only intercepted when pressed on their own. With `Ctrl` on top,
they pass straight through, so shortcuts like `Ctrl+=` (zoom in most apps) keep
working.

### Why cursor mode swallows bare keys but keeps shortcuts

Explicit request: in mouse mode no other key should work, "so there are no
mistakes". That way a stray `C` does not delete a word or fire a shortcut in the
app in front of you.

What is blocked is the **bare key**. Any combination carrying `Ctrl`, `Shift` or
`Alt` passes straight through, because that is what makes the tool usable with
both hands on one device: you select text with the mouse and then press `Ctrl+C`
to copy, `Ctrl+V` to paste. Modifiers on their own also always pass, since
swallowing them would make Windows think they are stuck and break every
combination.

Clicks work the same way. `Space` is always the left button, and the modifier you
are physically holding rides along with the click, exactly like a real mouse:

- `Shift`+`Space` — click **with Shift**: extend a selection from the last click
- `Ctrl`+`Space` — right button: context menu
- `Alt`+`Space` — middle button

Nothing is injected to fake the modifier: you really are holding Shift, and
Windows reads that state when the click lands, so the app sees a genuine
Shift+click.

So the full selection workflow is: click at the start, `Shift`+click at the end
(or drag), then `Ctrl+C`.

To get the previous behaviour back, run it with `-freekeys`.

### Why a red crosshair

A relative mouse can walk off the screen and disappear. The crosshair always shows
where the pointer is, plus two more safeguards:

- **Screen bounds**: the pointer never leaves the edges of the screen.
- **Corner jumps** with `Ctrl+Alt+Home` / `Ctrl+Alt+End`, to get back to a known
  spot in one keystroke.

The crosshair is a *layered* window with per-pixel alpha (see
[bug history](#bug-history-the-root-cause-of-each-problem)) so it does not swallow
clicks.

---

## Options

| Flag | Default | What it does |
| --- | --- | --- |
| `-speed` | `12` | pixels per step |
| `-turbo` | `60` | pixels per step with `Shift` |
| `-autoexit` | `45` | idle seconds before cursor mode switches off (`0` = never) |
| `-togglekey` | `Ctrl+Alt+M` | key or combination that toggles the mode |
| `-nocross` | off | hide the red crosshair |
|- `-nolegend` | off | do not show the key legend when cursor mode starts |
| `-freekeys` | off | let every key through in cursor mode |
| `-debug` | off | verbose logging (every key, every move) |
| `-version` | — | print the version and exit |
| `-selftest` | — | verify hook and `SendInput` without moving the cursor |
| `-install` / `-uninstall` | — | autostart management |

`-togglekey` accepts a single key (`F1`-`F24`, `Pause`, `ScrollLock`,
`CapsLock`, `Insert`, `Home`, `End`, `PageUp`, `PageDown`, `Space`, the arrow
keys or a letter) or a combination (`Ctrl+Alt+M`, `Shift+Space`). Since it
depends on the keyboard you happen to own, you can change it without touching
code:

```powershell
mouse-teclado.exe -togglekey F13
```

---

## Troubleshooting

The app writes `mouse-teclado.log` next to the `.exe`. It is the only way to see
what is happening when there is no terminal:

```
22:02:32 inicio: pid=7976 toggle=Ctrl+Alt+M speed=12 turbo=60 autoexit=45
22:02:32 hilo principal fijado con LockOSThread (OS thread=3104)
22:02:32 hook global de teclado instalado (hwnd=7537639)
22:02:32 bandeja creada: icono en el sistema, mensaje de bienvenida enviado
```

With `-debug` it also logs every key (`tecla vk=32`), every move
(`mueve dx=12 dy=0 step=12`) and what the ticker sees every 500 ms.

**After rebuilding, run `mouse-teclado.exe -install` again**: the hard link in the
Startup folder still points at the old `.exe` once the binary is replaced.

### Test tools

Three small programs in `%TEMP%` were used to verify this without a mouse:

| Folder | What it does |
| --- | --- |
| `clicksink` | A window that records the clicks it receives; proves a virtual click reaches a real app |
| `sendkey2` | Injects real keystrokes with `SendInput` and reads the app state from its log |
| `modestrict` | Checks that mouse mode swallows every key that is not a mouse key |

Two things worth knowing if you use them:

- The test window has to be **on top**: if something else covers it, it receives
  no `WM_MOUSEMOVE` at all and the diagnosis lies to you.
- **`SendInput` with `VK_ESCAPE` never reaches a low-level hook**: the call
  returns 1 (event delivered) but no hook ever sees `vk=27`, not even when sending
  the scan code explicitly. Some keys simply cannot be verified by injection.

---

## Bug history: the root cause of each problem

These four cost real time to find. They are documented because the code looks
strange without the reasoning, and running into them again would be a waste.

### 1. The hook never fired at all

**Symptom:** the app started, the log said "hook installed", but no key ever got
through.

**Cause:** a low-level hook is registered on one Windows thread and receives its
events **on that same thread**. Go moves goroutines between threads whenever it
feels like it, so the message loop could end up pumping a queue that was not the
hook's.

**Fix:** `runtime.LockOSThread()` at the top of the main goroutine.

### 2. Arrows worked, letters did not

**Symptom:** with the hook finally working, arrow keys moved the pointer but
`W A S D` did nothing.

**Cause:** key state was read with `GetAsyncKeyState` from the ticker goroutine.
That async state does not see injected keys and can drift between threads.

**Fix:** the hook itself fills `pressedKeys` on every keydown/keyup it sees, and
the ticker reads from there. Modifiers (`Shift`, `Ctrl`) still come from
`GetAsyncKeyState` on purpose: they are the ones most likely to get stuck when a
keyup never arrives.

### 3. Clicks did not register and the button got stuck

**Symptom:** the pointer moved but no click ever reached an app.

**Cause:** holding `Space` triggers keyboard auto-repeat: dozens of `WM_KEYDOWN`
with the repeat flag and a single `WM_KEYUP`. Every keydown sent another "button
down", so the left button stayed **stuck pressed** in Windows and every
subsequent click was swallowed.

**Fix:** only the first keydown counts (`Flags & 1`), a keydown is ignored while
the button is already down, and a watchdog releases the button if the key is no
longer held.

### 4. The crosshair ate every click

**Symptom:** with the crosshair visible, no click reached the app underneath.
Without it (`-nocross`), everything worked.

**Cause:** the crosshair was a normal `WS_POPUP` window sitting on top of the
pointer. Any window like that takes the clicks: movement is global, but the click
goes to whatever window is on top.

**Fix:** a *layered* window drawn into a 32bpp DIB with per-pixel alpha — only
the crosshair pixels have alpha 255, the rest is transparent and Windows lets the
click through. `WS_EX_TRANSPARENT` and `WM_NCHITTEST`/`HTTRANSPARENT` are there as
reinforcement, not as the solution.

Extra bit that cost time: `UpdateLayeredWindow` does **not** make the window
visible on its own, you have to call `ShowWindow` afterwards or the crosshair
exists but is invisible.

---

## Known limitations

- **Elevated windows:** if the app in front runs as administrator, Windows blocks
  clicks into it (UIPI protection). The pointer still moves, but the click is not
  registered. The fix is not running those apps elevated.
- **UAC secure desktop:** it is a different desktop session and cannot be driven.
- **Precision:** movement happens in 15 ms steps. It feels smooth, but for exact
  aiming hold `Z` (1 px steps).
- **Autostart uses the Startup folder, not the registry:** on this machine,
  registry writes made by child processes are silently discarded and can even
  crash the process. The Startup folder always works.

---

## Build

```powershell
go build -ldflags "-H windowsgui -s -w" -o mouse-teclado.exe .
```

`-H windowsgui` is what removes the console window: without it the `.exe` opens a
terminal.

After building, remember to reinstall autostart (`mouse-teclado.exe -install`).

---

## Code map

| File | Responsibility |
| --- | --- |
| `main.go` | Flags, startup, single instance, self-test, installation |
| `hook.go` | Global keyboard hook (`WH_KEYBOARD_LL`): what each key does |
| `mouse.go` | Movement, clicks, scroll, screen bounds, key state |
| `indicator.go` | Click-through red crosshair (layered window + DIB) |
| `tray.go` | System tray and menu |
| `win32.go` | Windows API declarations (structs, procs, constants) |

All movement goes through a 15 ms ticker (`ticker` in `main.go`), which reads the
key state assembled by the hook.