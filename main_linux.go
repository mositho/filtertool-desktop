//go:build linux && linuxdesktop

package main

import (
	"github.com/mositho/filtertool-desktop/internal/updates"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func preparePlatform() {}

func platformOptions(string) application.Options {
	return application.Options{Linux: application.LinuxOptions{ApplicationID: "io.github.mositho.filtertool-desktop"}}
}

// The automatic updater currently installs Windows executables only.
func platformUpdates(_ *application.App, version string, _ bool) (*updates.Manager, error) {
	return updates.New(nil, version, false)
}

func updateInstalledVersion(string) error { return nil }
