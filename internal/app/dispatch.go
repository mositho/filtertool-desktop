package app

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	c "github.com/mositho/filtertool-desktop/internal/compiler"
	"github.com/mositho/filtertool-desktop/internal/data"
)

// Dispatch is the single Wails-bound application service. It is transport-only:
// all persistent state lives in the Store, not in the webview or working folder.
func (s *Store) Dispatch(action string, input Object) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if input == nil {
		input = Object{}
	}
	name := c.Text(input["name"])
	if s.TemplateMode {
		switch action {
		case "createFilter", "duplicateFilter", "renameFilter", "deleteFilter", "importJSON", "importFiles", "importFolder", "saveDefaults", "saveStyles":
			return nil, errors.New("the template workspace only edits the bundled templates")
		case "saveConfig":
			if _, ok := data.Load[Object]("templates.json")[name]; !ok {
				return nil, errors.New("not a bundled template")
			}
		}
	}
	switch action {
	case "listFilters":
		filters, err := s.filters()
		return Object{"filters": filters, "templateMode": s.TemplateMode}, err
	case "getConfig":
		return s.readFilter(name)
	case "saveConfig":
		if _, err := s.readFilter(name); err != nil {
			return nil, err
		}
		cfg := c.Obj(input["config"])
		if err := s.writeFilter(name, cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "createFilter":
		if err := s.available(name, ""); err != nil {
			return nil, err
		}
		cfg := Object{"buildProfile": Object{}, "buildSpecificOptions": Object{}}
		if template := c.Text(input["template"]); template != "" {
			value, ok := data.Load[Object]("templates.json")[template]
			if !ok {
				return nil, errors.New("unknown template")
			}
			cfg = c.Obj(value)
		}
		return Object{"name": name}, s.writeFilter(name, cfg)
	case "duplicateFilter":
		newName := c.Text(input["newName"])
		if err := s.available(newName, ""); err != nil {
			return nil, err
		}
		cfg, err := s.readFilter(name)
		if err != nil {
			return nil, err
		}
		return Object{"name": newName}, s.writeFilter(newName, cfg)
	case "renameFilter":
		newName := c.Text(input["newName"])
		if err := s.available(newName, name); err != nil {
			return nil, err
		}
		cfg, err := s.readFilter(name)
		if err != nil {
			return nil, err
		}
		if name == newName {
			return Object{"name": name}, nil
		}
		// Case-only rename is safe through a temporary filename on Windows.
		old, _ := s.filterPath(name)
		newPath, _ := s.filterPath(newName)
		if strings.EqualFold(name, newName) {
			temp := old + ".rename"
			if err = os.Rename(old, temp); err != nil {
				return nil, err
			}
			if err = os.Rename(temp, newPath); err != nil {
				_ = os.Rename(temp, old)
				return nil, err
			}
		} else {
			if err = s.writeFilter(newName, cfg); err != nil {
				return nil, err
			}
			if err = os.Remove(old); err != nil {
				return nil, err
			}
		}
		return Object{"name": newName}, nil
	case "deleteFilter":
		path, err := s.filterPath(name)
		if err != nil {
			return nil, err
		}
		if _, err = s.readFilter(name); err != nil {
			return nil, err
		}
		if input["deleteGameFile"] == true {
			settings, err := s.settings()
			if err != nil {
				return nil, err
			}
			if folder := c.Text(settings["filterPath"]); folder != "" {
				err = os.Remove(filepath.Join(folder, exportName(name)))
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					return nil, err
				}
			}
		}
		return Object{"ok": true}, os.Remove(path)
	case "getSettings":
		return s.settings()
	case "saveSettings":
		settings := c.Clone(c.Obj(input["settings"]))
		tts := c.Obj(settings["tts"])
		speed := c.Num(tts["speed"], 1.6)
		if speed < 0.5 || speed > 2 {
			return nil, errors.New("TTS speed must be between 0.5 and 2")
		}
		locale := c.Text(tts["locale"])
		if !contains(c.Strings(data.Load[Object]("reference.json")["ttsLocales"]), locale) {
			return nil, errors.New("unsupported TTS locale")
		}
		folder := c.Text(settings["soundsFolder"])
		if folder != "" && !safeFile(folder) {
			return nil, errors.New("sound folder must be a single folder name")
		}
		if path := c.Text(settings["filterPath"]); path != "" && !filepath.IsAbs(path) {
			return nil, errors.New("choose an absolute Path of Exile filter folder")
		}
		return settings, s.write("settings.json", settings)
	case "getDefaults", "getStyles":
		d, st, err := s.globals()
		if err != nil {
			return nil, err
		}
		if action == "getDefaults" {
			u, err := s.optional("defaults.json")
			return Object{"defaults": d, "baseDefaults": data.Load[Object]("defaults.json"), "userDefaults": u}, err
		}
		u, err := s.optional("styles.json")
		return Object{"styles": st, "baseStyles": data.Load[Object]("styles.json"), "userStyles": u}, err
	case "saveDefaults", "saveStyles":
		key := "defaults"
		if action == "saveStyles" {
			key = "styles"
		}
		value := c.Obj(input[key])
		if err := validateOverrides(key, value); err != nil {
			return nil, err
		}
		return Object{"ok": true}, s.write(key+".json", value)
	case "reference":
		ref := data.Load[Object]("reference.json")
		_, styles, err := s.globals()
		if err != nil {
			return nil, err
		}
		if name != "" {
			cfg, err := s.readFilter(name)
			if err != nil {
				return nil, err
			}
			styles = c.Merge(styles, c.Obj(cfg["localStyles"]))
		}
		ref["styles"] = styles
		ref["sounds"] = s.sounds()
		return ref, nil
	case "filterDefaults":
		cfg, err := s.readFilter(name)
		if err != nil {
			return nil, err
		}
		d, _, err := s.globals()
		return Object{"defaults": c.Merge(d, c.Obj(cfg["localDefaults"]))}, err
	case "preview":
		cfg, err := s.configOrSaved(input)
		if err != nil {
			return nil, err
		}
		compiler, err := s.compiler(cfg)
		if err != nil {
			return nil, err
		}
		output, err := compiler.Compile(cfg)
		return Object{"output": output}, err
	case "exportFilter":
		return s.export(input)
	case "exportAllFilters":
		filters, err := s.filters()
		if err != nil {
			return nil, err
		}
		failures := []Object{}
		exported := 0
		for _, filter := range filters {
			n := c.Text(filter["name"])
			if _, err = s.export(Object{"name": n}); err != nil {
				failures = append(failures, Object{"name": n, "error": err.Error()})
			} else {
				exported++
			}
		}
		return Object{"exported": exported, "total": len(filters), "errors": failures}, nil
	case "share":
		return s.share(input)
	case "saveTemplateBundle":
		if !s.TemplateMode {
			return nil, errors.New("template bundle export is only available in the template workspace")
		}
		if s.SaveShareFile == nil {
			return nil, errors.New("native save dialog is unavailable")
		}
		bundle := Object{}
		for template := range data.Load[Object]("templates.json") {
			cfg, err := s.readFilter(template)
			if err != nil {
				return nil, err
			}
			if err = c.Validate(cfg); err != nil {
				return nil, err
			}
			bundle[template] = cfg
		}
		text, err := json.MarshalIndent(bundle, "", "  ")
		if err != nil {
			return nil, err
		}
		return s.SaveShareFile("templates.json", string(text)+"\n")
	case "saveShareFile":
		if s.SaveShareFile == nil {
			return nil, errors.New("native save dialog is unavailable")
		}
		result, err := s.share(input)
		if err != nil {
			return nil, err
		}
		return s.SaveShareFile(name+".json", c.Text(result["text"]))
	case "importJSON":
		return s.importJSON(name, c.Obj(input["value"]), nil, nil)
	case "importFiles":
		if s.ChooseFiles == nil {
			return nil, errors.New("native file picker is unavailable")
		}
		paths, err := s.ChooseFiles()
		if err != nil {
			return nil, err
		}
		return s.importFiles(paths)
	case "importFolder":
		if s.ChooseFolder == nil {
			return nil, errors.New("native folder picker is unavailable")
		}
		root, err := s.ChooseFolder("")
		if err != nil {
			return nil, err
		}
		if root == "" {
			return Object{"imported": []string{}, "errors": []Object{}}, nil
		}
		return s.importFolder(root)
	case "chooseFilterPath":
		if s.ChooseFolder == nil {
			return nil, errors.New("native folder picker is unavailable")
		}
		return s.ChooseFolder(c.Text(input["path"]))
	case "sounds":
		return Object{"sounds": s.sounds()}, nil
	case "importSound":
		if s.ChooseSoundFile == nil {
			return nil, errors.New("native file picker is unavailable")
		}
		path, err := s.ChooseSoundFile()
		if err != nil || path == "" {
			return "", err
		}
		return s.copyCustomSound(path)
	case "sound":
		id := c.Text(input["id"])
		if !safeFile(id) {
			return nil, errors.New("invalid sound name")
		}
		path := filepath.Join(s.Root, "sounds", id+".mp3")
		if input["game"] == true {
			path = filepath.Join(s.Root, "cache", "game-sounds", id+".mp3")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, errors.New("sound is not available; generate sounds first")
		}
		return "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString(b), nil
	case "generateSounds":
		return s.generateAll()
	case "syncSounds":
		return s.syncSounds()
	case "clipboardRead":
		if s.ClipboardRead == nil {
			return nil, errors.New("clipboard unavailable")
		}
		return s.ClipboardRead()
	case "clipboardWrite":
		if s.ClipboardWrite == nil {
			return nil, errors.New("clipboard unavailable")
		}
		return Object{"ok": true}, s.ClipboardWrite(c.Text(input["text"]))
	default:
		return nil, fmt.Errorf("unknown operation %q", action)
	}
}
func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
func safeFile(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\:\x00\r\n\"") && !strings.HasSuffix(name, ".") && !strings.HasSuffix(name, " ")
}
func validateOverrides(key string, value Object) error {
	// Reuse structural/range checks without treating settings as a filter.
	cfg := Object{"buildProfile": Object{}, "buildSpecificOptions": value}
	if key == "styles" {
		cfg["localStyles"] = value
	}
	return c.Validate(cfg)
}
