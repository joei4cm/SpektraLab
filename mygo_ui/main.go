// FilmPrint Lab — MyGo native UI for the SpektraLab Windows/Linux host.
//
// The Android Compose MVP was a thin throwaway screen. This rebuild follows
// MyGo's native-UI model (view = function of state, GPU-drawn, no webview)
// and the desktop editor layout: left film/paper rail | canvas | top actions.
// It talks to spektralab-host over HOST-PROTOCOL.md (no cgo into the engine).
//
//	cd mygo_ui && go run .                  # needs SPEKTRALAB_HOST + resources
//	SPEKTRALAB_MOCK_HOST=1 go run .         # TypeScript mock host, no Vulkan
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/joei4cm/SpektraLab/mygo_ui/host"
)

func main() {
	ctx := context.Background()
	repo := repoRoot()

	var (
		client *host.Client
		err    error
	)
	if os.Getenv("SPEKTRALAB_MOCK_HOST") == "1" {
		client, err = host.StartMock(ctx, repo)
		must(err)
		log.Println("using desktop/mock-host (SPEKTRALAB_MOCK_HOST=1)")
	} else {
		bin := os.Getenv("SPEKTRALAB_HOST")
		if bin == "" {
			bin = "spektralab-host"
		}
		resources := os.Getenv("SPEKTRALAB_ENGINE_RESOURCES")
		if resources == "" {
			resources = filepath.Join(repo, "engine", "resources")
			// Prefer a staged host tree when present.
			if staged := filepath.Join(repo, "build", "host-linux-x64", "engine"); dirExists(staged) {
				resources = staged
			}
		}
		client, err = host.Start(ctx, bin, resources)
		if err != nil {
			log.Fatalf("start host %q (resources %q): %v\n"+
				"Set SPEKTRALAB_HOST / SPEKTRALAB_ENGINE_RESOURCES, or SPEKTRALAB_MOCK_HOST=1.",
				bin, resources, err)
		}
	}
	defer func() { _ = client.Shutdown() }()

	hello, err := client.Hello()
	must(err)

	ed := newEditor(client, hello)
	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{
			Title:   "FilmPrint Lab",
			Width:   1280,
			Height:  800,
			Content: ui.View(ed.view),
		})
		ed.setWindow(w)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func repoRoot() string {
	if r := os.Getenv("SPEKTRALAB_REPO"); r != "" {
		return r
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	// mygo_ui/ → repo root
	if filepath.Base(wd) == "mygo_ui" {
		return filepath.Dir(wd)
	}
	return wd
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
