package main

import (
	"fmt"
	"image"
	"log"
	"path/filepath"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/joei4cm/SpektraLab/mygo_ui/host"
)

// editor is the window's state. The view is a pure function of it (MyGo's
// native-UI rule); host work runs on goroutines and lands back through
// win.Update so the main thread redraws.
type editor struct {
	win *mygo.Window
	h   *host.Client

	sidebarW float32

	film  Stock
	paper Stock

	path      string
	session   string
	status    string
	busy      atomic.Bool
	hostInfo  string
	nativeW   int
	nativeH   int
	preview   *ui.Bitmap
	previewW  int
	previewH  int
	errFlash  string
}

func newEditor(h *host.Client, hello host.Hello) *editor {
	return &editor{
		h:        h,
		sidebarW: 280,
		film:     defaultFilm(),
		paper:    defaultPaper(),
		status:   "Open a JPEG, PNG or RAW to begin.",
		hostInfo: fmt.Sprintf("host %s · protocol %d", hello.HostVersion, hello.Protocol),
	}
}

func (e *editor) setWindow(w *mygo.Window) { e.win = w }

func (e *editor) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Children(func() {
		e.topBar(c)
		ui.Split(c, &e.sidebarW, func() { e.leftRail(c) }, func() { e.canvas(c) }).
			SplitReset(280).Grow(1)
		ui.Row(c).FillWidth().Padding(8, 6).Background(t.Surface).Children(func() {
			msg := e.status
			if e.errFlash != "" {
				msg = e.errFlash
			}
			ui.Text(c, msg).FontSize(12).TextColor(t.TextMuted).Grow(1)
			ui.Text(c, e.hostInfo).FontSize(11).TextColor(t.TextMuted)
		})
	})
}

func (e *editor) topBar(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).FillWidth().Padding(12, 8).Gap(8).AlignItems(ui.Center).Background(t.Surface).Children(func() {
		ui.Text(c, "FilmPrint Lab").Bold().FontSize(15)
		ui.Text(c, "· MyGo").FontSize(12).TextColor(t.TextMuted)
		ui.Spacer(c)
		busy := e.busy.Load()
		if ui.Button(c.Key("open"), "Open…").Disabled(busy).Clicked() {
			e.pickOpen()
		}
		if ui.PrimaryButton(c.Key("render"), "Render").Disabled(busy || e.session == "").Clicked() {
			e.renderAsync()
		}
		if ui.Button(c.Key("export"), "Export PNG…").Disabled(busy || e.preview == nil).Clicked() {
			e.pickExport()
		}
	})
}

func (e *editor) leftRail(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(12).Gap(10).Background(t.Surface).Children(func() {
		ui.Text(c, "Film").Bold().FontSize(12).TextColor(t.TextMuted)
		e.stockList(c, films, e.film, func(s Stock) {
			if !canPrint(s, e.paper) {
				// Match colour/B&W paper when polarity flips.
				for _, p := range papers {
					if canPrint(s, p) {
						e.paper = p
						break
					}
				}
			}
			e.film = s
			e.applyStocksAsync()
		})
		ui.Text(c, "Paper").Bold().FontSize(12).TextColor(t.TextMuted)
		e.stockList(c, papers, e.paper, func(s Stock) {
			if !canPrint(e.film, s) {
				return
			}
			e.paper = s
			e.applyStocksAsync()
		})
	})
}

func (e *editor) stockList(c *ui.Context, items []Stock, selected Stock, onPick func(Stock)) {
	t := c.Theme()
	ui.Column(c).Gap(2).Children(func() {
		for _, s := range items {
			s := s
			label := s.Label
			if s.Monochrome {
				label += "  · B&W"
			}
			row := ui.Row(c.Key(s.ID)).FillWidth().Padding(8, 6).Gap(6).Radius(6)
			if s.ID == selected.ID {
				row.Background(t.Accent)
			} else {
				row.Background(t.SurfaceHover)
			}
			row.Children(func() {
				txt := ui.Text(c, label).FontSize(13).Grow(1)
				if s.ID == selected.ID {
					txt.TextColor(t.AccentText)
				}
			})
			if row.Clicked() {
				onPick(s)
			}
		}
	})
}

func (e *editor) canvas(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Center().Background(ui.Hex("#1c1c1e")).Children(func() {
		switch {
		case e.busy.Load() && e.preview == nil:
			ui.Text(c, "Working…").FontSize(14).TextColor(t.TextMuted)
		case e.preview != nil:
			ui.Image(c, e.preview).Fit(ui.Contain).
				MaxWidth(float32(e.previewW)).MaxHeight(float32(e.previewH))
			ui.Textf(c, "%s  ·  %d×%d → %d×%d  ·  %s / %s",
				filepath.Base(e.path), e.nativeW, e.nativeH, e.previewW, e.previewH,
				e.film.Label, e.paper.Label).
				FontSize(12).TextColor(t.TextMuted).Padding(0, 8)
		default:
			ui.Text(c, "No frame open").FontSize(16).TextColor(t.TextMuted)
			ui.Text(c, "Open an image, choose film + paper, then Render.").
				FontSize(13).TextColor(t.TextMuted)
		}
	})
}

func (e *editor) pickOpen() {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:   e.win,
		Title:    "Open image",
		Multiple: false,
		Filters: []mygo.FileFilter{
			{Name: "Images", MIMETypes: []string{"image/*"}},
			{Name: "RAW + raster", Extensions: []string{
				"jpg", "jpeg", "png", "tif", "tiff",
				"arw", "nef", "nrw", "dng", "cr2", "cr3", "raf", "rw2", "orf", "pef", "srw",
			}},
		},
	})
	if err != nil || len(paths) == 0 {
		return
	}
	e.openAsync(paths[0])
}

func (e *editor) pickExport() {
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      e.win,
		Title:       "Export PNG",
		DefaultPath: "filmprint.png",
		Filters:     []mygo.FileFilter{{Name: "PNG", Extensions: []string{"png"}}},
	})
	if err != nil || path == "" {
		return
	}
	e.exportAsync(path)
}

func (e *editor) params() map[string]any {
	return map[string]any{
		"film_stock":  e.film.ID,
		"print_stock": e.paper.ID,
	}
}

func (e *editor) openAsync(path string) {
	if !e.busy.CompareAndSwap(false, true) {
		return
	}
	e.errFlash = ""
	e.status = "Opening " + filepath.Base(path) + "…"
	go func() {
		defer e.busy.Store(false)
		if e.session != "" {
			_ = e.h.CloseSession(e.session)
		}
		open, err := e.h.Open(path, e.params())
		e.win.Update(func() {
			if err != nil {
				e.errFlash = err.Error()
				e.status = "Open failed."
				return
			}
			e.path = path
			e.session = open.Session
			e.nativeW, e.nativeH = open.Width, open.Height
			e.preview = nil
			e.status = fmt.Sprintf("Opened %s (%d×%d). Rendering…", filepath.Base(path), open.Width, open.Height)
		})
		if err != nil {
			return
		}
		e.renderLocked()
	}()
}

func (e *editor) applyStocksAsync() {
	if e.session == "" || !e.busy.CompareAndSwap(false, true) {
		return
	}
	e.status = fmt.Sprintf("Applying %s → %s…", e.film.Label, e.paper.Label)
	go func() {
		defer e.busy.Store(false)
		if err := e.h.SetParams(e.session, e.params()); err != nil {
			e.win.Update(func() { e.errFlash = err.Error() })
			return
		}
		e.renderLocked()
	}()
}

func (e *editor) renderAsync() {
	if e.session == "" || !e.busy.CompareAndSwap(false, true) {
		return
	}
	e.status = "Rendering…"
	go func() {
		defer e.busy.Store(false)
		e.renderLocked()
	}()
}

// renderLocked runs a preview render; caller owns busy.
func (e *editor) renderLocked() {
	img, err := e.h.RenderRGBA8(e.session, "preview", 1600)
	e.win.Update(func() {
		if err != nil {
			e.errFlash = err.Error()
			e.status = "Render failed."
			return
		}
		bmp := rgbaToBitmap(img)
		e.preview = bmp
		e.previewW, e.previewH = img.Width, img.Height
		e.errFlash = ""
		e.status = fmt.Sprintf("Rendered %d×%d · %s / %s", img.Width, img.Height, e.film.Label, e.paper.Label)
	})
}

func (e *editor) exportAsync(path string) {
	if e.session == "" || !e.busy.CompareAndSwap(false, true) {
		return
	}
	e.status = "Exporting…"
	go func() {
		defer e.busy.Store(false)
		err := e.h.ExportPNG(e.session, path)
		e.win.Update(func() {
			if err != nil {
				e.errFlash = err.Error()
				e.status = "Export failed."
				return
			}
			e.status = "Exported " + filepath.Base(path)
		})
	}()
}

func rgbaToBitmap(img host.Image) *ui.Bitmap {
	rgba := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))
	copy(rgba.Pix, img.Data)
	return ui.NewBitmap(rgba)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
