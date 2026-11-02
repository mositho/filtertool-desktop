//go:build windows

package main

import (
	"path/filepath"

	"github.com/mositho/filtertool-desktop/internal/desktop"
	"github.com/mositho/filtertool-desktop/internal/updates"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

func preparePlatform() { updater.HandleHelperMode() }

func platformOptions(root string) application.Options {
	return application.Options{Windows: application.WindowsOptions{WebviewUserDataPath: filepath.Join(root, "cache", "webview2")}}
}

func platformUpdates(native *application.App, version string, enabled bool) (*updates.Manager, error) {
	return updates.New(native.Updater, version, enabled)
}

func updateInstalledVersion(version string) error { return desktop.UpdateInstalledVersion(version) }
