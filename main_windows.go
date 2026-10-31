//go:build windows

package main

import (
	"embed"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/mositho/filtertool-desktop/internal/app"
	"github.com/mositho/filtertool-desktop/internal/bundle"
	"github.com/mositho/filtertool-desktop/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist branding/logo.png
var assets embed.FS

type App struct {
	store  *app.Store
	native *application.App
	dirty  atomic.Bool
}

func (a *App) Dispatch(action string, input map[string]any) (any, error) {
	if action == "setDirty" {
		a.dirty.Store(input["dirty"] == true)
		return nil, nil
	}
	if action == "close" {
		a.dirty.Store(false)
		a.native.Quit()
		return nil, nil
	}
	return a.store.Dispatch(action, input)
}
func main() {
	devMode := os.Getenv("FILTERTOOL_PROFILE") == "dev"
	for _, arg := range os.Args[1:] {
		if arg == "--dev" {
			devMode = true
		}
	}
	root, err := app.UserDataDir()
	if err != nil {
		log.Fatal(err)
	}
	var store *app.Store
	if devMode {
		sourceDir := os.Getenv("FILTERTOOL_SOURCE_DIR")
		if sourceDir == "" {
			sourceDir, err = filepath.Abs(filepath.Join("internal", "data"))
			if err != nil {
				log.Fatal(err)
			}
		}
		store, err = app.OpenDev(root, sourceDir)
	} else {
		store, err = app.Open(root)
	}
	if err != nil {
		log.Fatal(err)
	}
	root = store.Root
	processor, err := bundle.Prepare(root)
	if err != nil {
		log.Fatal(err)
	}
	store.AudioProcessor = processor
	service := &App{store: store}
	icon, _ := assets.ReadFile("branding/logo.png")
	var window *application.WebviewWindow
	title := "Filtertool"
	instanceID := "io.github.mositho.filtertool-desktop"
	if devMode {
		title += " — Dev Editor"
		instanceID += ".dev"
	}
	native := application.New(application.Options{Name: title, Description: "Path of Exile leveling filter builder", Icon: icon, Services: []application.Service{application.NewService(service)}, Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)}, Windows: application.WindowsOptions{WebviewUserDataPath: filepath.Join(root, "cache", "webview2")}, SingleInstance: &application.SingleInstanceOptions{UniqueID: instanceID, OnSecondInstanceLaunch: func(application.SecondInstanceData) {
		if window != nil {
			window.Show()
			window.Focus()
		}
	}}})
	service.native = native
	window = native.Window.NewWithOptions(application.WebviewWindowOptions{Title: title, Width: 1440, Height: 960, MinWidth: 1024, MinHeight: 720, BackgroundColour: application.NewRGB(10, 10, 10), URL: "/"})
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if service.dirty.Load() {
			e.Cancel()
			native.Event.Emit("close-requested")
		}
	})
	store.ChooseFolder = func(path string) (string, error) {
		return desktop.DialogSelection(native.Dialog.OpenFile().SetTitle("Choose folder").CanChooseFiles(false).CanChooseDirectories(true).CanCreateDirectories(true).SetDirectory(path).AttachToWindow(window).PromptForSingleSelection())
	}
	store.ChooseFiles = func() ([]string, error) {
		return desktop.DialogSelection(native.Dialog.OpenFile().SetTitle("Import Filtertool JSON").CanChooseFiles(true).CanChooseDirectories(false).AddFilter("Filtertool JSON", "*.json").AttachToWindow(window).PromptForMultipleSelection())
	}
	store.ChooseSoundFile = func() (string, error) {
		return desktop.DialogSelection(native.Dialog.OpenFile().SetTitle("Choose custom sound").AddFilter("MP3 audio", "*.mp3").AttachToWindow(window).PromptForSingleSelection())
	}
	store.SaveShareFile = func(name, text string) (string, error) {
		dialog := native.Dialog.SaveFile()
		dialog.SetOptions(&application.SaveFileDialogOptions{Title: "Save shared filter", Filename: name, CanCreateDirectories: true, Window: window, Filters: []application.FileFilter{{DisplayName: "Filtertool JSON", Pattern: "*.json"}}})
		path, err := desktop.DialogSelection(dialog.PromptForSingleSelection())
		if err != nil || path == "" {
			return path, err
		}
		return path, os.WriteFile(path, []byte(text), 0600)
	}
	store.Emit = func(name string, value any) { native.Event.Emit(name, value) }
	store.ClipboardRead = func() (string, error) {
		text, ok := native.Clipboard.Text()
		if !ok {
			return "", errors.New("could not read clipboard")
		}
		return text, nil
	}
	store.ClipboardWrite = func(text string) error {
		if !native.Clipboard.SetText(text) {
			return errors.New("could not copy to clipboard")
		}
		return nil
	}
	if err = native.Run(); err != nil {
		log.Fatal(err)
	}
}
