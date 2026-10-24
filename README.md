<img width="6016" height="1504" alt="head and logo flamtap" src="https://github.com/user-attachments/assets/1eeba479-c06e-4e5e-9333-6eaa60c51884" />

A lightweight Go converter with a Gio-based GUI that transforms General MIDI (GM) drum tracks on MIDI channel 10 into four-lane Pro Drums charts for Clone Hero.

## Platforms: Windows and Linux

## Features:

- GM drum MIDI conversion
- Four-lane Pro Drums chart generation
- Simple graphical interface built with Gio
- Lightweight implementation in Go

## Build & Usage:

### 1. Build

Build the project using the provided scripts:

- **Windows:** `build.bat`
- **Linux:** `./build.sh`

Alternatively, build manually using Go:

```bash
go mod tidy
```

**Windows:**
```bash
go build -ldflags "-H windowsgui" -o f1amtap.exe ./cmd/f1amtap
```

**Linux:**
```bash
go build -ldflags="-s -w" -o f1amtap ./cmd/f1amtap
```

### 2. Run

Launch `f1amtap.exe` on Windows or `f1amtap` on Linux.

In the GUI:
1. Select the input MIDI file containing the drum track you want to convert.
2. Specify the output directory.

The converter creates an output folder named after the input MIDI file.

### 3. Review the Chart

The converter generates a `notes.chart` file without a `song.ini`.

The generated chart is a starting point and may require manual adjustments before it is playable.

Open it in **Moonscraper Chart Editor** to review the timing and note grid, and make any necessary corrections.
