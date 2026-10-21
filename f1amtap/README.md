# f1amtap

MIDI drum tracks → Clone Hero / Moonscraper `.chart` (`[ExpertDrums]`).

Lightweight converter GM drum MIDI → 4-lane pro-drums chart.

## Layout (Go-style)

```text
f1amtap/
  cmd/f1amtap/main.go     # CLI + --gui entrypoint
  internal/gui/gui.go     # Gio UI (header, browse, start)
  chartgen/               # MIDI → chart conversion library
  assets/                 # optional header_bg.png, logo.png
  build.bat / build.sh    # build binary into project root
```

## Build

Binary is written to the **project root**:

| OS | Output |
|----|--------|
| Windows | `f1amtap.exe` |
| Linux / macOS | `f1amtap` |

```bash
# Windows
build.bat

# Linux / macOS
./build.sh

# or manually
go mod tidy
go build -ldflags "-H windowsgui" -o f1amtap.exe ./cmd/f1amtap   # Windows
go build -ldflags="-s -w" -o f1amtap ./cmd/f1amtap     # Unix
```

Requires Go **1.24.4+** (same Gio stack as Pata2Fla: `gioui.org v0.10.3`, `sqweek/dialog`).

```bat
set GOPROXY=https://proxy.golang.org,direct
```

## CLI

```bash
f1amtap input.mid
f1amtap -o D:\charts input.mid
f1amtap --gui
```

| Flag | Meaning |
|------|---------|
| `-o` | Output root (default: folder of the MIDI) |
| `-channel` | MIDI channel (default 10; `-1` = all) |
| `--gui` | Open the window |

Output: `<out>/<midi_stem>/notes.chart` (no `song.ini`).

## GUI

Slate + cyan: header images, Browse dialogs, full-width Start.

Optional assets (cwd = project root when you launch the binary from there):

```text
assets/header_bg.png
assets/logo.png
```

## Limitations

- Drum MIDI only (GM-ish pitches on channel 10)
- Review timing/grid in Moonscraper
