package gui

import (
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"f1amtap/chartgen"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/sqweek/dialog"
)

var (
	colorBgDark     = color.NRGBA{R: 15, G: 23, B: 42, A: 255}
	colorCardBg     = color.NRGBA{R: 30, G: 41, B: 59, A: 255}
	colorCardBorder = color.NRGBA{R: 51, G: 65, B: 85, A: 255}
	colorInputBg    = color.NRGBA{R: 15, G: 23, B: 42, A: 255}
	colorCyanAccent = color.NRGBA{R: 6, G: 182, B: 212, A: 255}
	colorTextLight  = color.NRGBA{R: 248, G: 250, B: 252, A: 255}
	colorTextMuted  = color.NRGBA{R: 148, G: 163, B: 184, A: 255}
	colorTextDim    = color.NRGBA{R: 100, G: 116, B: 139, A: 255}
	colorGreen      = color.NRGBA{R: 34, G: 197, B: 94, A: 255}
	colorRed        = color.NRGBA{R: 239, G: 68, B: 68, A: 255}
)

const (
	HeaderHeightDp    = 140
	HeaderMaxHeightDp = 200
	LogoWidthPercent  = 90
	LogoMaxHeightDp   = 0
)

type GUIState struct {
	mu     sync.Mutex
	window *app.Window

	headerBgImg image.Image
	headerBgOp  paint.ImageOp
	hasHeaderBg bool
	logoImg     image.Image
	logoOp      paint.ImageOp
	hasLogo     bool

	inputEditor         widget.Editor
	btnBrowseInput      widget.Clickable
	btnOpenInputFolder  widget.Clickable
	outputEditor        widget.Editor
	btnBrowseOutput     widget.Clickable
	btnOpenOutputFolder widget.Clickable
	btnStart            widget.Clickable

	busy     bool
	status   string
	statusOK bool
}

func loadFirstImage(candidates ...string) image.Image {
	for _, path := range candidates {
		if path == "" {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		img, _, err := image.Decode(f)
		_ = f.Close()
		if err == nil && img != nil {
			return img
		}
	}
	return nil
}

func newGUIState(w *app.Window) *GUIState {
	s := &GUIState{
		window: w,
		status: "Ready. Select a drum MIDI and click START CONVERSION.",
	}
	s.inputEditor.SingleLine = true
	s.inputEditor.Submit = true
	s.outputEditor.SingleLine = true
	s.outputEditor.Submit = true
	s.outputEditor.SetText("output")

	if bgImg := loadFirstImage(
		filepath.Join("assets", "flamtap_logo.png"),
		filepath.Join("assets", "flamtap_logo.jpg"),
		filepath.Join("assets", "background.png"),
		filepath.Join("assets", "background.jpg"),
		filepath.Join("assets", "bg.png"),
		"header_bg.png",
		"background.png",
	); bgImg != nil {
		s.headerBgImg = bgImg
		s.headerBgOp = paint.NewImageOp(bgImg)
		s.hasHeaderBg = true
	}

	if logoImg := loadFirstImage(
		filepath.Join("assets", "logo.png"),
		filepath.Join("assets", "f1amtap_logo.png"),
		filepath.Join("assets", "logo.jpg"),
		"logo.png",
	); logoImg != nil {
		s.logoImg = logoImg
		s.logoOp = paint.NewImageOp(logoImg)
		s.hasLogo = true
	}

	return s
}

func (s *GUIState) browseInputFile() {
	// Same pattern as Pata2Fla (github.com/sqweek/dialog).
	path, err := dialog.File().
		Filter("MIDI Files", "mid", "midi").
		Filter("All Files", "*").
		Title("Select Drum MIDI File").
		Load()
	if err != nil {
		return
	}
	s.mu.Lock()
	s.inputEditor.SetText(path)
	s.mu.Unlock()
	s.window.Invalidate()
}

func (s *GUIState) browseOutputDir() {
	path, err := dialog.Directory().
		Title("Select Output Directory").
		Browse()
	if err != nil {
		return
	}
	s.mu.Lock()
	s.outputEditor.SetText(path)
	s.mu.Unlock()
	s.window.Invalidate()
}

func (s *GUIState) openFolder(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = "."
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		dir = filepath.Dir(dir)
	}
	_ = os.MkdirAll(dir, 0755)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	_ = cmd.Start()
}

func (s *GUIState) runConvert() {
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return
	}
	midi := strings.TrimSpace(s.inputEditor.Text())
	out := strings.TrimSpace(s.outputEditor.Text())
	s.mu.Unlock()

	if midi == "" {
		s.mu.Lock()
		s.status = "Error: no MIDI file specified"
		s.statusOK = false
		s.mu.Unlock()
		s.window.Invalidate()
		return
	}
	if _, err := os.Stat(midi); err != nil {
		s.mu.Lock()
		s.status = "Error: MIDI file not found"
		s.statusOK = false
		s.mu.Unlock()
		s.window.Invalidate()
		return
	}
	if out == "" {
		out = filepath.Dir(midi)
		if out == "" {
			out = "."
		}
		s.mu.Lock()
		s.outputEditor.SetText(out)
		s.mu.Unlock()
	}

	s.mu.Lock()
	s.busy = true
	s.status = "Converting..."
	s.statusOK = false
	s.mu.Unlock()
	s.window.Invalidate()

	cfg := chartgen.DefaultConfig()
	path, err := chartgen.ConvertFile(midi, out, cfg)

	s.mu.Lock()
	s.busy = false
	if err != nil {
		s.status = "Error: " + err.Error()
		s.statusOK = false
	} else {
		s.status = "OK → " + path
		s.statusOK = true
	}
	s.mu.Unlock()
	s.window.Invalidate()
}

func layoutCard(gtx layout.Context, content layout.Widget) layout.Dimensions {
	radius := unit.Dp(8)
	return widget.Border{
		Color:        colorCardBorder,
		CornerRadius: radius,
		Width:        unit.Dp(1),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, colorCardBg, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(radius)).Op(gtx.Ops))
		return layout.UniformInset(unit.Dp(10)).Layout(gtx, content)
	})
}

func layoutInputBox(gtx layout.Context, th *material.Theme, ed *widget.Editor, hint string) layout.Dimensions {
	radius := unit.Dp(6)
	return widget.Border{
		Color:        colorCardBorder,
		CornerRadius: radius,
		Width:        unit.Dp(1),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, colorInputBg, clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(radius)).Op(gtx.Ops))
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			mEd := material.Editor(th, ed, hint)
			mEd.Color = colorTextLight
			mEd.HintColor = colorTextDim
			mEd.TextSize = unit.Sp(13)
			return mEd.Layout(gtx)
		})
	})
}

func layoutButton(gtx layout.Context, th *material.Theme, clk *widget.Clickable, txt string, bg, fg color.NRGBA) layout.Dimensions {
	btn := material.Button(th, clk, txt)
	btn.Background = bg
	btn.Color = fg
	btn.CornerRadius = unit.Dp(6)
	btn.Inset = layout.Inset{
		Top: unit.Dp(7), Bottom: unit.Dp(7),
		Left: unit.Dp(12), Right: unit.Dp(12),
	}
	btn.TextSize = unit.Sp(12)
	return btn.Layout(gtx)
}

func (s *GUIState) layoutHeader(gtx layout.Context, th *material.Theme) layout.Dimensions {
	radius := unit.Dp(8)
	calcHeight := gtx.Dp(unit.Dp(HeaderHeightDp))

	if s.hasHeaderBg && s.headerBgImg != nil {
		b := s.headerBgImg.Bounds()
		if b.Dx() > 0 && b.Dy() > 0 {
			ratio := float32(b.Dy()) / float32(b.Dx())
			bgHeight := int(float32(gtx.Constraints.Max.X) * ratio)
			calcHeight = (calcHeight + bgHeight) / 2
		}
	}

	minH := gtx.Dp(unit.Dp(HeaderHeightDp))
	maxH := gtx.Dp(unit.Dp(HeaderMaxHeightDp))
	if calcHeight < minH {
		calcHeight = minH
	}
	if calcHeight > maxH {
		calcHeight = maxH
	}

	headerSize := image.Point{X: gtx.Constraints.Max.X, Y: calcHeight}

	return widget.Border{
		Color:        colorCardBorder,
		CornerRadius: radius,
		Width:        unit.Dp(1),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = layout.Constraints{Min: headerSize, Max: headerSize}

		clipP := clip.UniformRRect(image.Rectangle{Max: headerSize}, gtx.Dp(radius)).Push(gtx.Ops)
		defer clipP.Pop()

		paint.FillShape(gtx.Ops, colorCardBg, clip.Rect{Max: headerSize}.Op())

		if s.hasHeaderBg {
			bg := widget.Image{
				Src:      s.headerBgOp,
				Fit:      widget.Cover,
				Position: layout.Center,
			}
			gBg := gtx
			gBg.Constraints = layout.Constraints{Min: headerSize, Max: headerSize}
			bg.Layout(gBg)
			paint.FillShape(gtx.Ops,
				color.NRGBA{R: 15, G: 23, B: 42, A: 70},
				clip.Rect{Max: headerSize}.Op())
		}

		if s.hasLogo && s.logoImg != nil {
			b := s.logoImg.Bounds()
			if b.Dx() > 0 && b.Dy() > 0 {
				logoRatio := float32(b.Dy()) / float32(b.Dx())
				maxLogoW := int(float32(headerSize.X) * float32(LogoWidthPercent) / 100)
				maxLogoH := int(float32(headerSize.Y) * float32(LogoWidthPercent) / 100)
				targetW := maxLogoW
				targetH := int(float32(targetW) * logoRatio)
				if targetH > maxLogoH {
					targetH = maxLogoH
					targetW = int(float32(targetH) / logoRatio)
				}
				if LogoMaxHeightDp > 0 {
					capH := gtx.Dp(unit.Dp(LogoMaxHeightDp))
					if targetH > capH {
						targetH = capH
						targetW = int(float32(targetH) / logoRatio)
					}
				}
				logoSize := image.Point{X: targetW, Y: targetH}
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gLogo := gtx
					gLogo.Constraints = layout.Constraints{Min: logoSize, Max: logoSize}
					return widget.Image{
						Src:      s.logoOp,
						Fit:      widget.Contain,
						Position: layout.Center,
					}.Layout(gLogo)
				})
			}
		}

		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(26), "")
			lbl.Color = colorCyanAccent
			return lbl.Layout(gtx)
		})
	})
}

func (s *GUIState) layoutInputCard(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layoutCard(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(12), "INPUT MIDI FILE (.mid)")
				lbl.Color = colorCyanAccent
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layoutInputBox(gtx, th, &s.inputEditor, "e.g. drums.mid or click Browse...")
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layoutButton(gtx, th, &s.btnBrowseInput, "Browse...", colorCyanAccent, color.NRGBA{R: 2, G: 6, B: 23, A: 255})
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layoutButton(gtx, th, &s.btnOpenInputFolder, "Folder", colorCardBorder, colorTextLight)
					}),
				)
			}),
		)
	})
}

func (s *GUIState) layoutOutputCard(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layoutCard(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(12), "OUTPUT DIRECTORY")
				lbl.Color = colorCyanAccent
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layoutInputBox(gtx, th, &s.outputEditor, "output")
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layoutButton(gtx, th, &s.btnBrowseOutput, "Browse...", colorCyanAccent, color.NRGBA{R: 2, G: 6, B: 23, A: 255})
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layoutButton(gtx, th, &s.btnOpenOutputFolder, "Folder", colorCardBorder, colorTextLight)
					}),
				)
			}),
		)
	})
}

func (s *GUIState) layoutActionCard(gtx layout.Context, th *material.Theme) layout.Dimensions {
	s.mu.Lock()
	busy := s.busy
	status := s.status
	ok := s.statusOK
	s.mu.Unlock()

	return layoutCard(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				txt := "▶  START CONVERSION"
				bg := colorCyanAccent
				fg := color.NRGBA{R: 2, G: 6, B: 23, A: 255}
				if busy {
					txt = "⏳  CONVERTING..."
					bg = colorCardBorder
					fg = colorTextDim
				}
				btn := material.Button(th, &s.btnStart, txt)
				btn.Background = bg
				btn.Color = fg
				btn.CornerRadius = unit.Dp(8)
				btn.Inset = layout.Inset{Top: unit.Dp(14), Bottom: unit.Dp(14), Left: unit.Dp(16), Right: unit.Dp(16)}
				btn.TextSize = unit.Sp(14)
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return btn.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				col := colorTextMuted
				if strings.HasPrefix(status, "Error") {
					col = colorRed
				} else if ok {
					col = colorGreen
				}
				lbl := material.Label(th, unit.Sp(12), status)
				lbl.Color = col
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(11), "Writes <output>/<midi_name>/notes.chart  ·  no song.ini")
				lbl.Color = colorTextDim
				return lbl.Layout(gtx)
			}),
		)
	})
}

func (s *GUIState) handleEvents(gtx layout.Context) {
	if s.btnBrowseInput.Clicked(gtx) {
		go s.browseInputFile()
	}
	if s.btnOpenInputFolder.Clicked(gtx) {
		s.openFolder(filepath.Dir(s.inputEditor.Text()))
	}
	if s.btnBrowseOutput.Clicked(gtx) {
		go s.browseOutputDir()
	}
	if s.btnOpenOutputFolder.Clicked(gtx) {
		s.openFolder(s.outputEditor.Text())
	}
	if s.btnStart.Clicked(gtx) {
		go s.runConvert()
	}
}

func (s *GUIState) layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	paint.Fill(gtx.Ops, colorBgDark)
	return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return s.layoutHeader(gtx, th)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return s.layoutInputCard(gtx, th)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return s.layoutOutputCard(gtx, th)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return s.layoutActionCard(gtx, th)
			}),
		)
	})
}

func run(w *app.Window) error {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	th.Palette.Fg = colorTextLight
	th.Palette.Bg = colorBgDark
	th.Palette.ContrastBg = colorCyanAccent
	th.Palette.ContrastFg = color.NRGBA{R: 2, G: 6, B: 23, A: 255}

	state := newGUIState(w)
	var ops op.Ops

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			state.handleEvents(gtx)
			state.layout(gtx, th)
			e.Frame(gtx.Ops)
		}
	}
}

// Run starts the Gio window (blocks until closed).
func Run() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("f1amtap — MIDI → Clone Hero chart"))
		w.Option(app.Size(unit.Dp(720), unit.Dp(520)))
		w.Option(app.MinSize(unit.Dp(560), unit.Dp(420)))
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
