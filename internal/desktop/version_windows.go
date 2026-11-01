package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Only an installed copy may change its Windows installed-apps version.
func UpdateInstalledVersion(version string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall\Filtertool`, registry.QUERY_VALUE|registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer key.Close()
	location, _, err := key.GetStringValue("InstallLocation")
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if strings.EqualFold(filepath.Clean(exe), filepath.Join(location, "Filtertool.exe")) {
		return key.SetStringValue("DisplayVersion", version)
	}
	return nil
}
