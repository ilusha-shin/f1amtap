package chartgen

import (
	"fmt"
	"sort"
	"strings"
)

type ChartNote struct {
	Tick int64
	Lane int
}

// formatChart writes a CH-style .chart. bpm is written into [SyncTrack] as B = bpm*1000.
func formatChart(notes []ChartNote, fullFile bool, name string, bpm float64, resolution int) string {
	sort.SliceStable(notes, func(i, j int) bool {
		if notes[i].Tick == notes[j].Tick {
			return notes[i].Lane > notes[j].Lane
		}
		return notes[i].Tick < notes[j].Tick
	})

	if name == "" {
		name = "Drum Export"
	}
	if bpm < 20 {
		bpm = 120
	}
	if resolution <= 0 {
		resolution = 192
	}
	// Moonscraper / CH: tempo event value ≈ BPM * 1000
	bVal := int(bpm*1000 + 0.5)

	var sb strings.Builder

	if fullFile {
		sb.WriteString("[Song]\n{\n")
		sb.WriteString(fmt.Sprintf("  Name = %q\n", name))
		sb.WriteString("  Offset = 0\n")
		sb.WriteString(fmt.Sprintf("  Resolution = %d\n", resolution))
		sb.WriteString("  Player2 = bass\n")
		sb.WriteString("  Difficulty = 0\n")
		sb.WriteString("  PreviewStart = 0\n")
		sb.WriteString("  PreviewEnd = 0\n")
		sb.WriteString("  Genre = \"rock\"\n")
		sb.WriteString("  MediaType = \"cd\"\n")
		sb.WriteString("}\n")
		sb.WriteString("[SyncTrack]\n{\n")
		sb.WriteString("  0 = TS 4\n")
		sb.WriteString(fmt.Sprintf("  0 = B %d\n", bVal))
		sb.WriteString("}\n")
		sb.WriteString("[Events]\n{\n}\n")
	}

	sb.WriteString("[ExpertDrums]\n{\n")
	for _, n := range notes {
		sb.WriteString(fmt.Sprintf("  %d = N %d 0\n", n.Tick, n.Lane))
	}
	sb.WriteString("}\n")

	return sb.String()
}
