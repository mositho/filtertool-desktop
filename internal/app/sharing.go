package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	c "github.com/mositho/filtertool-desktop/internal/compiler"
	"github.com/mositho/filtertool-desktop/internal/data"
)

// snapshotDifference includes absent sender values as null, so a recipient's
// extra cutoff/style setting cannot leak into a self-contained shared filter.
func snapshotDifference(value, base Object) Object {
	out := c.Difference(value, base)
	for k, b := range base {
		v, ok := value[k]
		if !ok {
			out[k] = nil
			continue
		}
		if _, ok := b.(map[string]any); ok {
			if _, ok := v.(map[string]any); ok {
				d := snapshotDifference(c.Obj(v), c.Obj(b))
				if len(d) > 0 {
					out[k] = d
				}
			}
		}
	}
	return out
}
func normalizedStyles(styles Object) Object {
	out := c.Clone(styles)
	for k, v := range out {
		style := c.Obj(v)
		for field, value := range (Object{"size": float64(45), "text": "#C8C8C8", "background": "#000000", "border": "#000000"}) {
			if _, ok := style[field]; !ok {
				style[field] = value
			}
		}
		out[k] = style
	}
	return out
}
func (s *Store) share(input Object) (Object, error) {
	cfg, err := s.configOrSaved(input)
	if err != nil {
		return nil, err
	}
	if err = c.Validate(cfg); err != nil {
		return nil, err
	}
	cfg = c.Clone(cfg)
	var value any = cfg
	if input["includeGlobals"] == true {
		d, st, err := s.globals()
		if err != nil {
			return nil, err
		}
		value = Object{"format": "filtertool", "config": cfg, "defaults": c.Merge(d, c.Obj(cfg["localDefaults"])), "styles": normalizedStyles(c.Merge(st, c.Obj(cfg["localStyles"])))}
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	text := strings.TrimSuffix(buffer.String(), "\n")
	return Object{"text": text, "characters": len([]rune(text))}, nil
}
func (s *Store) imported(value Object, defaults, styles Object) (Object, error) {
	cfg := value
	if format, ok := value["format"]; ok {
		if format != "filtertool" {
			return nil, errors.New("unsupported sharing format")
		}
		cfg = c.Clone(c.Obj(value["config"]))
		d, st, err := s.globals()
		if err != nil {
			return nil, err
		}
		cfg["localDefaults"] = snapshotDifference(c.Obj(value["defaults"]), d)
		cfg["localStyles"] = snapshotDifference(normalizedStyles(c.Obj(value["styles"])), normalizedStyles(st))
	} else {
		cfg = c.Clone(cfg)
		if defaults != nil || styles != nil {
			d, st, err := s.globals()
			if err != nil {
				return nil, err
			}
			cfg["localDefaults"] = snapshotDifference(c.Merge(data.Load[Object]("defaults.json"), defaults), d)
			cfg["localStyles"] = snapshotDifference(normalizedStyles(c.Merge(data.Load[Object]("styles.json"), styles)), normalizedStyles(st))
		}
	}
	// JSON copied by the old web app can omit one of the two empty objects.
	if cfg["buildProfile"] == nil {
		cfg["buildProfile"] = Object{}
	}
	if cfg["buildSpecificOptions"] == nil {
		cfg["buildSpecificOptions"] = Object{}
	}
	if _, ok := value["format"]; !ok && len(value) > 0 {
		if _, profile := value["buildProfile"]; !profile {
			if _, options := value["buildSpecificOptions"]; !options {
				return nil, errors.New("expected a Filtertool JSON filter")
			}
		}
	}
	if err := c.Validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
func (s *Store) importJSON(name string, value, defaults, styles Object) (Object, error) {
	if err := s.available(name, ""); err != nil {
		return nil, err
	}
	cfg, err := s.imported(value, defaults, styles)
	if err != nil {
		return nil, err
	}
	if err = s.writeFilter(name, cfg); err != nil {
		return nil, err
	}
	return Object{"name": name}, nil
}
func (s *Store) importFiles(paths []string) (Object, error) {
	imported := []string{}
	failures := []Object{}
	for _, path := range paths {
		value, err := readObject(path)
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if filepath.Base(path) == "config.json" {
			name = filepath.Base(filepath.Dir(path))
		}
		if err == nil {
			_, err = s.importJSON(name, value, nil, nil)
		}
		if err != nil {
			failures = append(failures, Object{"name": name, "error": err.Error()})
		} else {
			imported = append(imported, name)
		}
	}
	return Object{"imported": imported, "errors": failures}, nil
}
func (s *Store) importFolder(root string) (Object, error) {
	// Only read the old app's documented JSON locations. Never execute or
	// recursively discover config.ts files, and never modify the source folder.
	filtersRoot := filepath.Join(root, "src", "filters")
	if _, err := os.Stat(filtersRoot); err != nil {
		return nil, errors.New("choose the old Filtertool project folder")
	}
	d, err := readObject(filepath.Join(filtersRoot, "shared", "user-defaults.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	st, err := readObject(filepath.Join(filtersRoot, "shared", "user-styles.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	entries, err := os.ReadDir(filtersRoot)
	if err != nil {
		return nil, err
	}
	imported := []string{}
	failures := []Object{}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "shared" || entry.Name() == "template" {
			continue
		}
		file := filepath.Join(filtersRoot, entry.Name(), "config.json")
		value, err := readObject(file)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err == nil {
			var cfg Object
			cfg, err = s.imported(value, d, st)
			if err == nil {
				err = s.available(entry.Name(), "")
			}
			if err == nil {
				err = s.importLegacySounds(cfg, filepath.Join(root, "sounds"))
			}
			if err == nil {
				err = s.writeFilter(entry.Name(), cfg)
			}
		}
		if err != nil {
			failures = append(failures, Object{"name": entry.Name(), "error": fmt.Sprint(err)})
		} else {
			imported = append(imported, entry.Name())
		}
	}
	return Object{"imported": imported, "errors": failures}, nil
}
