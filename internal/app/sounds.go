package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	c "github.com/mositho/filtertool-desktop/internal/compiler"
	"github.com/mositho/filtertool-desktop/internal/data"
)

func (s *Store) copyCustomSound(path string) (string, error) {
	name := filepath.Base(path)
	if !safeFile(name) || !strings.EqualFold(filepath.Ext(name), ".mp3") {
		return "", errors.New("choose an MP3 sound file")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > 20*1024*1024 {
		return "", errors.New("choose a regular MP3 file smaller than 20 MB")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	name = strings.TrimSuffix(name, filepath.Ext(name)) + ".mp3"
	for _, entry := range data.Load[[]Object]("manifest.json") {
		if strings.EqualFold(name, c.Text(entry["id"])+".mp3") {
			hash := sha256.Sum256(b)
			name = "custom-" + hex.EncodeToString(hash[:6]) + ".mp3"
			break
		}
	}
	// Do not overwrite a sound already used by another filter.
	target := filepath.Join(s.Root, "sounds", name)
	if existing, err := os.ReadFile(target); err == nil {
		if bytes.Equal(existing, b) {
			return name, nil
		}
		hash := sha256.Sum256(b)
		name = "custom-" + hex.EncodeToString(hash[:6]) + ".mp3"
		target = filepath.Join(s.Root, "sounds", name)
	}
	if err = writeAtomic(target, b); err != nil {
		return "", err
	}
	return name, nil
}
func (s *Store) importLegacySounds(cfg Object, source string) error {
	var walk func(any) error
	walk = func(value any) error {
		switch v := value.(type) {
		case map[string]any:
			for key, item := range v {
				if key == "soundFileName" && c.Text(item) != "" {
					name := c.Text(item)
					if !safeFile(name) {
						return errors.New("custom sound must be a filename")
					}
					copied, err := s.copyCustomSound(filepath.Join(source, name))
					if err != nil {
						return fmt.Errorf("custom sound %q: %w", name, err)
					}
					v[key] = copied
				} else if err := walk(item); err != nil {
					return err
				}
			}
		case []any:
			for _, item := range v {
				if err := walk(item); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(cfg)
}
func customSoundFiles(value any) map[string]bool {
	out := map[string]bool{}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, item := range x {
				if k == "soundFileName" {
					out[c.Text(item)] = true
				} else {
					walk(item)
				}
			}
		case []any:
			for _, item := range x {
				walk(item)
			}
		}
	}
	walk(value)
	return out
}

func (s *Store) sounds() []Object {
	out := []Object{}
	for _, entry := range data.Load[[]Object]("manifest.json") {
		id := c.Text(entry["id"])
		_, err := os.Stat(filepath.Join(s.Root, "sounds", id+".mp3"))
		out = append(out, Object{"id": id, "name": entry["name"], "hasFile": err == nil})
	}
	return out
}
func (s *Store) generateFile(filename, text string, settings Object) error {
	if !safeFile(filename) || !strings.HasSuffix(filename, ".mp3") {
		return errors.New("invalid sound filename")
	}
	if len([]rune(text)) > 200 {
		return errors.New("TTS phrases must not exceed 200 characters")
	}
	tts := c.Obj(settings["tts"])
	locale := c.Text(tts["locale"])
	speed := c.Num(tts["speed"], 1.6)
	signature := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%g|processing-v2", text, locale, speed)))
	hash := hex.EncodeToString(signature[:])
	state, err := s.optional("sound-state.json")
	if err != nil {
		return err
	}
	target := filepath.Join(s.Root, "sounds", filename)
	if _, err = os.Stat(target); err == nil && state[filename] == hash {
		return nil
	}
	if s.AudioProcessor == nil {
		return errors.New("FFmpeg was not found on PATH. Install FFmpeg and make sure ffmpeg.exe is available on PATH to generate TTS sounds, then restart Filtertool.")
	}
	fetch := s.FetchAudio
	if fetch == nil {
		fetch = fetchAudio
	}
	b, err := fetch(text, locale)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Join(s.Root, "cache"), "tts-*.mp3")
	if err != nil {
		return err
	}
	temp := f.Name()
	f.Close()
	defer os.Remove(temp)
	if err = s.AudioProcessor(b, temp, speed); err != nil {
		return err
	}
	processed, err := os.ReadFile(temp)
	if err != nil {
		return err
	}
	if err = writeAtomic(target, processed); err != nil {
		return err
	}
	state[filename] = hash
	return s.write("sound-state.json", state)
}
func fetchAudio(text, locale string) ([]byte, error) {
	u := "https://translate.google.com/translate_tts?" + url.Values{"ie": {"UTF-8"}, "q": {text}, "tl": {locale}, "total": {"1"}, "idx": {"0"}, "textlen": {fmt.Sprint(len([]rune(text)))}, "client": {"tw-ob"}, "prev": {"input"}, "ttsspeed": {"1"}}.Encode()
	client := http.Client{Timeout: 30 * time.Second}
	res, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("could not generate %q: %w", text, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TTS service returned HTTP %d for %q", res.StatusCode, text)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, 5*1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(b) > 5*1024*1024 {
		return nil, errors.New("TTS response too large")
	}
	return b, nil
}
func (s *Store) generateAll() (Object, error) {
	settings, err := s.settings()
	if err != nil {
		return nil, err
	}
	entries := data.Load[[]Object]("manifest.json")
	filters, err := s.filters()
	if err != nil {
		return nil, err
	}
	total := len(entries) + len(filters)
	done := 0
	for _, entry := range entries {
		text := c.Text(entry["text"])
		if s.Emit != nil {
			s.Emit("sound-progress", Object{"done": done, "total": total, "text": text})
		}
		if err = s.generateFile(c.Text(entry["id"])+".mp3", text, settings); err != nil {
			return nil, err
		}
		done++
	}
	for _, filter := range filters {
		name := c.Text(filter["name"])
		cfg, err := s.readFilter(name)
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
		if s.Emit != nil {
			s.Emit("sound-progress", Object{"done": done, "total": total, "text": name})
		}
		if err = s.prepareOutputSounds(output, settings, cfg); err != nil {
			return nil, err
		}
		done++
	}
	if err = s.copySounds(settings); err != nil {
		return nil, err
	}
	if s.Emit != nil {
		s.Emit("sound-progress", Object{"done": done, "total": done, "text": ""})
	}
	return Object{"generated": done}, nil
}
func (s *Store) prepareOutputSounds(output string, settings, cfg Object) error {
	manifest := map[string]string{}
	for _, entry := range data.Load[[]Object]("manifest.json") {
		manifest[c.Text(entry["id"])+".mp3"] = c.Text(entry["text"])
	}
	seen := map[string]bool{}
	custom := customSoundFiles(cfg)
	d, _, err := s.globals()
	if err != nil {
		return err
	}
	for name := range customSoundFiles(d) {
		custom[name] = true
	}
	matches := soundReference.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		ref := match[1]
		prefix := soundFolder(settings) + "/"
		if !strings.HasPrefix(ref, prefix) {
			return errors.New("custom sound must be inside the sound pack")
		}
		file := strings.TrimPrefix(ref, prefix)
		if !safeFile(file) {
			return errors.New("custom sound must use a filename, not a nested path")
		}
		if seen[file] {
			continue
		}
		seen[file] = true
		text, known := manifest[file]
		if custom[file] {
			if _, err := os.Stat(filepath.Join(s.Root, "sounds", file)); err != nil {
				return fmt.Errorf("custom sound %q is missing; choose it with Browse", file)
			}
			continue
		}
		if !known {
			text = strings.ReplaceAll(strings.TrimSuffix(file, ".mp3"), "_", " ")
		}
		if err := s.generateFile(file, text, settings); err != nil {
			return err
		}
	}
	return nil
}
