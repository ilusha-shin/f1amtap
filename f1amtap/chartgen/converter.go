package chartgen

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	LaneKick       = 0
	LaneRed        = 1
	LaneYellow     = 2
	LaneBlue       = 3
	LaneGreen      = 4
	LaneDoubleKick = 32
	CymbalModifier = 64
)

type Config struct {
	TargetResolution int     // chart Resolution (default 192)
	StartTick        int64   // lead-in ticks before first note
	Channel          int     // MIDI channel 1–16; 10 = drums; -1 = all
	Auto2xKick       bool
	MaxKickInterval  int64
	AutoToms         bool
	FullFile         bool
	BPM              float64 // if 0, try MIDI tempo, else 120
	Quantize         int     // 16, 24, or 32 (grid); 0 = no extra snap beyond resolution
	Name             string  // [Song] Name
}

func DefaultConfig() Config {
	return Config{
		TargetResolution: 192,
		StartTick:        0,
		Channel:          10,
		Auto2xKick:       true,
		MaxKickInterval:  96,
		AutoToms:         true,
		FullFile:         true,
		BPM:              0, // MIDI tempo if present, else 120
		Quantize:         0, // no extra grid snap
		Name:             "",
	}
}

type DrumHit struct {
	Lane     int
	IsCymbal bool
}

type DrumConverter struct {
	cfg               Config
	kickNotes         map[uint8]bool
	redNotes          map[uint8]bool
	yellowTomNotes    map[uint8]bool
	blueTomNotes      map[uint8]bool
	greenTomNotes     map[uint8]bool
	yellowCymbalNotes map[uint8]bool
	blueCymbalNotes   map[uint8]bool
	greenCymbalNotes  map[uint8]bool
	ignoredNotes      map[uint8]bool
}

func NewConverter(cfg Config) *DrumConverter {
	return &DrumConverter{
		cfg:               cfg,
		kickNotes:         makeMap(35, 36),
		redNotes:          makeMap(38, 39, 40),
		yellowTomNotes:    makeMap(48, 50),
		blueTomNotes:      makeMap(45, 47, 56),
		greenTomNotes:     makeMap(41, 43),
		yellowCymbalNotes: makeMap(42, 44, 46, 54, 55, 95),
		blueCymbalNotes:   makeMap(51, 52, 53, 59),
		greenCymbalNotes:  makeMap(49, 57),
		ignoredNotes:      makeMap(0, 37, 58),
	}
}

func makeMap(notes ...uint8) map[uint8]bool {
	m := make(map[uint8]bool, len(notes))
	for _, n := range notes {
		m[n] = true
	}
	return m
}

// ConvertFile reads a MIDI path and writes notes.chart into outDir/<stem>/notes.chart.
// Returns the written chart path.
func ConvertFile(midiPath, outDir string, cfg Config) (string, error) {
	f, err := os.Open(midiPath)
	if err != nil {
		return "", fmt.Errorf("open midi: %w", err)
	}
	defer f.Close()

	stem := strings.TrimSuffix(filepath.Base(midiPath), filepath.Ext(midiPath))
	if stem == "" {
		stem = "drums"
	}
	if cfg.Name == "" {
		cfg.Name = stem
	}

	conv := NewConverter(cfg)
	chart, err := conv.Convert(f)
	if err != nil {
		return "", err
	}

	destDir := filepath.Join(outDir, stem)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	outPath := filepath.Join(destDir, "notes.chart")
	if err := os.WriteFile(outPath, []byte(chart), 0644); err != nil {
		return "", fmt.Errorf("write chart: %w", err)
	}
	return outPath, nil
}

func (c *DrumConverter) Convert(r io.Reader) (string, error) {
	midiData, err := parseMIDI(r)
	if err != nil {
		return "", fmt.Errorf("midi parse error: %w", err)
	}

	var filtered []NoteEvent
	usedNotes := make(map[uint8]bool)
	for _, ev := range midiData.Events {
		if c.cfg.Channel == -1 || int(ev.Channel) == c.cfg.Channel {
			filtered = append(filtered, ev)
			usedNotes[ev.Note] = true
		}
	}

	if len(filtered) == 0 {
		// Fallback: some exports put drums on channel 1
		if c.cfg.Channel == 10 {
			for _, ev := range midiData.Events {
				filtered = append(filtered, ev)
				usedNotes[ev.Note] = true
			}
		}
		if len(filtered) == 0 {
			return "", fmt.Errorf("no drum notes found on channel %d", c.cfg.Channel)
		}
	}

	if c.cfg.AutoToms {
		c.optimizeToms(usedNotes)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].AbsoluteTick < filtered[j].AbsoluteTick
	})

	ppq := float64(midiData.TimeDivision)
	if ppq <= 0 {
		ppq = 480
	}
	res := c.cfg.TargetResolution
	if res <= 0 {
		res = 192
	}
	ratio := ppq / float64(res)

	firstMidiTick := int64(math.Round(float64(filtered[0].AbsoluteTick) / ratio))

	tickMap := make(map[int64][]uint8)
	var uniqueTicks []int64

	for _, ev := range filtered {
		tick := int64(math.Round(float64(ev.AbsoluteTick)/ratio)) - firstMidiTick + c.cfg.StartTick
		tick = quantizeTick(tick, int64(res), c.cfg.Quantize)
		if _, exists := tickMap[tick]; !exists {
			uniqueTicks = append(uniqueTicks, tick)
		}
		tickMap[tick] = append(tickMap[tick], ev.Note)
	}
	sort.Slice(uniqueTicks, func(i, j int) bool { return uniqueTicks[i] < uniqueTicks[j] })

	var chartEvents []ChartNote
	var lastKickTick int64 = -1
	lastWasDouble := true

	for _, tick := range uniqueTicks {
		notes := tickMap[tick]
		var kickHit *DrumHit
		var handHits []DrumHit

		for _, note := range notes {
			if c.ignoredNotes[note] {
				continue
			}

			if c.kickNotes[note] {
				lane := LaneKick
				if c.cfg.Auto2xKick {
					if !lastWasDouble && lastKickTick >= 0 && (lastKickTick+c.cfg.MaxKickInterval >= tick) {
						lane = LaneDoubleKick
						lastWasDouble = true
					} else {
						lastWasDouble = false
					}
				}
				lastKickTick = tick
				kickHit = &DrumHit{Lane: lane, IsCymbal: false}
			} else if c.redNotes[note] {
				handHits = append(handHits, DrumHit{Lane: LaneRed, IsCymbal: false})
			} else if c.yellowCymbalNotes[note] {
				handHits = append(handHits, DrumHit{Lane: LaneYellow, IsCymbal: true})
			} else if c.blueCymbalNotes[note] {
				handHits = append(handHits, DrumHit{Lane: LaneBlue, IsCymbal: true})
			} else if c.greenCymbalNotes[note] {
				handHits = append(handHits, DrumHit{Lane: LaneGreen, IsCymbal: true})
			} else if c.yellowTomNotes[note] {
				handHits = append(handHits, DrumHit{Lane: LaneYellow, IsCymbal: false})
			} else if c.blueTomNotes[note] {
				handHits = append(handHits, DrumHit{Lane: LaneBlue, IsCymbal: false})
			} else if c.greenTomNotes[note] {
				handHits = append(handHits, DrumHit{Lane: LaneGreen, IsCymbal: false})
			}
		}

		if kickHit != nil {
			chartEvents = append(chartEvents, ChartNote{Tick: tick, Lane: kickHit.Lane})
		}

		processedHands := c.processHandHits(handHits)
		for _, h := range processedHands {
			if h.IsCymbal {
				chartEvents = append(chartEvents, ChartNote{Tick: tick, Lane: h.Lane + CymbalModifier})
			}
			chartEvents = append(chartEvents, ChartNote{Tick: tick, Lane: h.Lane})
		}
	}

	bpm := c.cfg.BPM
	if bpm <= 0 && midiData.TempoBPM > 0 {
		bpm = midiData.TempoBPM
	}
	if bpm <= 0 {
		bpm = 120
	}

	name := c.cfg.Name
	return formatChart(chartEvents, c.cfg.FullFile, name, bpm, res), nil
}

// quantizeTick snaps tick to a grid derived from chart resolution and quantize denominator.
// quantize 16 → 1/16 grid, 24 → 1/24 (mixed/triplet-friendly), 32 → 1/32. 0 = no snap.
func quantizeTick(tick, res int64, quantize int) int64 {
	if quantize <= 0 || res <= 0 {
		return tick
	}
	grid := res * 4 / int64(quantize) // e.g. res=192, q=16 → 48; q=24 → 32; q=32 → 24
	if grid < 1 {
		grid = 1
	}
	return ((tick + grid/2) / grid) * grid
}

func (c *DrumConverter) processHandHits(hits []DrumHit) []DrumHit {
	if len(hits) == 0 {
		return nil
	}
	if len(hits) == 1 {
		return hits
	}

	h1 := hits[0]
	h2 := hits[1]

	if h1.IsCymbal && h2.IsCymbal && (h1.Lane == h2.Lane) {
		if h1.Lane == LaneGreen {
			h1.Lane = LaneYellow
		} else {
			h1.Lane = LaneGreen
		}
	}

	if h1.Lane == h2.Lane && h1.IsCymbal == h2.IsCymbal {
		switch h1.Lane {
		case LaneRed:
			h2.Lane = LaneYellow
		case LaneYellow:
			h2.Lane = LaneBlue
		case LaneBlue:
			h2.Lane = LaneGreen
		case LaneGreen:
			h2.Lane = LaneYellow
		}
	}

	return []DrumHit{h1, h2}
}

func (c *DrumConverter) optimizeToms(usedNotes map[uint8]bool) {
	var toms []uint8
	for i := uint8(35); i <= 81; i++ {
		if c.yellowTomNotes[i] || c.blueTomNotes[i] || c.greenTomNotes[i] {
			if usedNotes[i] {
				toms = append(toms, i)
			}
		}
	}

	sort.Slice(toms, func(i, j int) bool { return toms[i] < toms[j] })

	c.yellowTomNotes = make(map[uint8]bool)
	c.blueTomNotes = make(map[uint8]bool)
	c.greenTomNotes = make(map[uint8]bool)

	switch len(toms) {
	case 1:
		c.blueTomNotes[toms[0]] = true
	case 2:
		c.greenTomNotes[toms[0]] = true
		c.blueTomNotes[toms[1]] = true
	case 3:
		c.greenTomNotes[toms[0]] = true
		c.blueTomNotes[toms[1]] = true
		c.yellowTomNotes[toms[2]] = true
	case 4:
		c.greenTomNotes[toms[0]] = true
		c.greenTomNotes[toms[1]] = true
		c.blueTomNotes[toms[2]] = true
		c.yellowTomNotes[toms[3]] = true
	case 5:
		c.greenTomNotes[toms[0]] = true
		c.greenTomNotes[toms[1]] = true
		c.blueTomNotes[toms[2]] = true
		c.blueTomNotes[toms[3]] = true
		c.yellowTomNotes[toms[4]] = true
	default:
		if len(toms) >= 6 {
			c.greenTomNotes[toms[0]] = true
			c.greenTomNotes[toms[1]] = true
			c.blueTomNotes[toms[2]] = true
			c.blueTomNotes[toms[3]] = true
			c.yellowTomNotes[toms[4]] = true
			c.yellowTomNotes[toms[5]] = true
		}
	}
}
