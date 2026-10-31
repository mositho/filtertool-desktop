package app

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	c "github.com/mositho/filtertool-desktop/internal/compiler"
	"github.com/mositho/filtertool-desktop/internal/data"
)

var soundReference = regexp.MustCompile(`CustomAlertSound "([^"]+)"`)

func (s *Store) export(input Object) (Object, error) {
	name := c.Text(input["name"])
	if _, err := s.readFilter(name); err != nil {
		return nil, err
	}
	cfg, err := s.configOrSaved(input)
	if err != nil {
		return nil, err
	}
	settings, err := s.settings()
	if err != nil {
		return nil, err
	}
	folder := c.Text(settings["filterPath"])
	if folder == "" {
		return nil, errors.New("choose your Path of Exile filter folder first")
	}
	compiler, err := s.compiler(cfg)
	if err != nil {
		return nil, err
	}
	output, err := compiler.Compile(cfg)
	if err != nil {
		return nil, err
	}
	if err = s.writeFilter(name, cfg); err != nil {
		return nil, err
	}
	if err = s.prepareOutputSounds(output, settings, cfg); err != nil {
		return nil, err
	}
	if err = s.copySounds(settings); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(folder, 0755); err != nil {
		return nil, err
	}
	filename := exportName(name)
	if err = writeAtomic(filepath.Join(folder, filename), []byte(output)); err != nil {
		return nil, err
	}
	return Object{"fileName": filename, "filterPath": folder, "soundFolder": soundFolder(settings)}, nil
}
func (s *Store) copySounds(settings Object) error {
	game := c.Text(settings["filterPath"])
	if game == "" {
		return errors.New("choose your Path of Exile filter folder first")
	}
	target := filepath.Join(game, soundFolder(settings))
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(s.Root, "sounds"))
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.HasSuffix(e.Name(), ".mp3") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.Root, "sounds", e.Name()))
		if err != nil {
			return err
		}
		if err = writeAtomic(filepath.Join(target, e.Name()), b); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) syncSounds() (Object, error) {
	settings, err := s.settings()
	if err != nil {
		return nil, err
	}
	keep := map[string]bool{}
	for _, entry := range data.Load[[]Object]("manifest.json") {
		keep[c.Text(entry["id"])+".mp3"] = true
	}
	filters, err := s.filters()
	if err != nil {
		return nil, err
	}
	for _, filter := range filters {
		cfg, err := s.readFilter(c.Text(filter["name"]))
		if err != nil {
			return nil, err
		}
		compiler, err := s.compiler(cfg)
		if err != nil {
			return nil, err
		}
		output, err := compiler.Compile(cfg)
		if err != nil {
			return nil, err
		}
		for _, match := range soundReference.FindAllStringSubmatch(output, -1) {
			keep[strings.TrimPrefix(match[1], soundFolder(settings)+"/")] = true
		}
	}
	// Remove only app-generated, unreferenced cache files. Never delete custom
	// user audio or arbitrary files in the game's filter directory.
	state, err := s.optional("sound-state.json")
	if err != nil {
		return nil, err
	}
	removed := 0
	for name := range state {
		if keep[name] || !safeFile(name) {
			continue
		}
		if err = os.Remove(filepath.Join(s.Root, "sounds", name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		delete(state, name)
		removed++
	}
	if err = s.write("sound-state.json", state); err != nil {
		return nil, err
	}
	if err = s.copySounds(settings); err != nil {
		return nil, err
	}
	return Object{"removed": removed}, nil
}
