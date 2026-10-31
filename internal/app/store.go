// Package app owns desktop data and operations independently of the Wails window.
package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	c "github.com/mositho/filtertool-desktop/internal/compiler"
	"github.com/mositho/filtertool-desktop/internal/data"
)

type Object = map[string]any
type Store struct {
	Root            string
	mu              sync.Mutex
	ChooseFolder    func(string) (string, error)
	ChooseFiles     func() ([]string, error)
	ChooseSoundFile func() (string, error)
	Emit            func(string, any)
	AudioProcessor  func([]byte, string, float64) error
	FetchAudio      func(string, string) ([]byte, error)
	SaveShareFile   func(string, string) (string, error)
	ClipboardRead   func() (string, error)
	ClipboardWrite  func(string) error
}

func UserDataDir() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "Filtertool"), nil
}
func Open(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("data directory is required")
	}
	s := &Store{Root: root}
	for _, dir := range []string{"filters", "sounds", "cache"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			return nil, err
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".initialized")); errors.Is(err, os.ErrNotExist) {
		filters, err := s.filters()
		if err != nil {
			return nil, err
		}
		if len(filters) == 0 {
			for name, value := range data.Load[Object]("examples.json") {
				if err = s.writeFilter(name, c.Obj(value)); err != nil {
					return nil, err
				}
			}
		}
		if _, err = os.Stat(filepath.Join(root, "settings.json")); errors.Is(err, os.ErrNotExist) {
			if err = s.write("settings.json", Object{"filterPath": DetectFilterPath(), "soundsFolder": "", "tts": Object{"locale": "en-US", "speed": 1.6}}); err != nil {
				return nil, err
			}
		}
		if err = writeAtomic(filepath.Join(root, ".initialized"), []byte("Filtertool\n")); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if _, err := s.settings(); err != nil {
		return nil, err
	}
	return s, nil
}
func writeAtomic(path string, b []byte) error {
	if st, err := os.Lstat(path); err == nil && st.Mode()&os.ModeSymlink != 0 {
		return errors.New("refusing to replace a symbolic link")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".filtertool-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func (s *Store) write(name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.Root, name), append(b, '\n'))
}
func readObject(path string) (Object, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > 2*1024*1024 {
		return nil, errors.New("expected a regular JSON file smaller than 2 MB")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v Object
	if err = json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", filepath.Base(path), err)
	}
	if v == nil {
		return nil, errors.New("expected a JSON object")
	}
	return v, nil
}
func (s *Store) optional(name string) (Object, error) {
	m, err := readObject(filepath.Join(s.Root, name))
	if errors.Is(err, os.ErrNotExist) {
		return Object{}, nil
	}
	return m, err
}
func (s *Store) settings() (Object, error) { return readObject(filepath.Join(s.Root, "settings.json")) }

var validName = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9 _-]{0,79}$`)

func checkName(name string) error {
	if !validName.MatchString(name) || strings.TrimSpace(name) != name {
		return errors.New("use 1–80 letters, numbers, spaces, underscores or hyphens")
	}
	upper := strings.ToUpper(name)
	for _, reserved := range []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"} {
		if upper == reserved {
			return errors.New("that name is reserved by Windows")
		}
	}
	return nil
}
func (s *Store) filterPath(name string) (string, error) {
	if err := checkName(name); err != nil {
		return "", err
	}
	return filepath.Join(s.Root, "filters", name+".json"), nil
}
func (s *Store) readFilter(name string) (Object, error) {
	p, err := s.filterPath(name)
	if err != nil {
		return nil, err
	}
	return readObject(p)
}
func (s *Store) writeFilter(name string, cfg Object) error {
	if err := c.Validate(cfg); err != nil {
		return err
	}
	p, err := s.filterPath(name)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(p, append(b, '\n'))
}
func (s *Store) filters() ([]Object, error) {
	entries, err := os.ReadDir(filepath.Join(s.Root, "filters"))
	if err != nil {
		return nil, err
	}
	out := []Object{}
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".json") {
			name := strings.TrimSuffix(e.Name(), ".json")
			if checkName(name) == nil {
				out = append(out, Object{"name": name, "hasConfigJson": true})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(c.Text(out[i]["name"])) < strings.ToLower(c.Text(out[j]["name"]))
	})
	return out, nil
}
func exportName(name string) string {
	words := regexp.MustCompile(`[^a-z0-9]+`).Split(strings.ToLower(name), -1)
	out := ""
	for _, w := range words {
		if w != "" {
			out += strings.ToUpper(w[:1]) + w[1:]
		}
	}
	if out == "" {
		out = "Filter"
	}
	return out + ".filter"
}
func (s *Store) available(name, except string) error {
	if err := checkName(name); err != nil {
		return err
	}
	filters, err := s.filters()
	if err != nil {
		return err
	}
	for _, f := range filters {
		other := c.Text(f["name"])
		if other == except {
			continue
		}
		if strings.EqualFold(other, name) || strings.EqualFold(exportName(other), exportName(name)) {
			return errors.New("a filter with that name or exported filename already exists")
		}
	}
	return nil
}
func (s *Store) globals() (Object, Object, error) {
	d, err := s.optional("defaults.json")
	if err != nil {
		return nil, nil, err
	}
	st, err := s.optional("styles.json")
	if err != nil {
		return nil, nil, err
	}
	return c.Merge(data.Load[Object]("defaults.json"), d), c.Merge(data.Load[Object]("styles.json"), st), nil
}
func (s *Store) compiler(cfg Object) (*c.Compiler, error) {
	d, st, err := s.globals()
	if err != nil {
		return nil, err
	}
	settings, err := s.settings()
	if err != nil {
		return nil, err
	}
	return c.New(c.Merge(d, c.Obj(cfg["localDefaults"])), c.Merge(st, c.Obj(cfg["localStyles"])), soundFolder(settings)), nil
}
func soundFolder(settings Object) string {
	folder := c.Text(settings["soundsFolder"])
	if folder == "" {
		return "poeft-sounds-v2"
	}
	return folder
}
func (s *Store) configOrSaved(input Object) (Object, error) {
	if cfg, ok := input["config"].(map[string]any); ok {
		return cfg, nil
	}
	return s.readFilter(c.Text(input["name"]))
}
