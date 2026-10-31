//go:build windows

package app

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
)

func DetectFilterPath() string {
	documents, err := windows.KnownFolderPath(windows.FOLDERID_Documents, 0)
	if err != nil {
		home, _ := os.UserHomeDir()
		documents = filepath.Join(home, "Documents")
	}
	return filepath.Join(documents, "My Games", "Path of Exile")
}
