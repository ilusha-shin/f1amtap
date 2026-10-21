package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"f1amtap/chartgen"
	"f1amtap/internal/gui"
)

func main() {
	guiFlag := flag.Bool("gui", false, "open Gio GUI")
	outDir := flag.String("o", "", "output directory (default: next to MIDI)")
	channel := flag.Int("channel", 10, "MIDI channel (10=drums, -1=all)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "f1amtap — MIDI drums → Clone Hero .chart\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  f1amtap [flags] input.mid\n  f1amtap --gui\n\n")
		fmt.Fprintf(os.Stderr, "Output: <out>/<midi_stem>/notes.chart\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *guiFlag || (flag.NArg() == 0 && len(os.Args) == 1) {
		gui.Run()
		return
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	midiPath := flag.Arg(0)
	cfg := chartgen.DefaultConfig()
	cfg.Channel = *channel

	baseOut := *outDir
	if baseOut == "" {
		baseOut = filepath.Dir(midiPath)
		if baseOut == "" {
			baseOut = "."
		}
	}

	outPath, err := chartgen.ConvertFile(midiPath, baseOut, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK → %s\n", outPath)
}
