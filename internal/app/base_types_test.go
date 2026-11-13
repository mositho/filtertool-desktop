package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	c "github.com/mositho/filtertool-desktop/internal/compiler"
	"github.com/mositho/filtertool-desktop/internal/data"
)

func TestOldFilterPreviewAndReexportsUseCurrentBaseTypes(t *testing.T) {
	s := testStore(t)
	game := t.TempDir()
	settings, err := s.settings()
	if err != nil {
		t.Fatal(err)
	}
	settings["filterPath"] = game
	mustCall(t, s, "saveSettings", Object{"settings": settings})
	s.FetchAudio = func(string, string) ([]byte, error) { return []byte("test TTS"), nil }
	s.AudioProcessor = func(b []byte, path string, _ float64) error { return os.WriteFile(path, b, 0600) }
	changes := data.Load[struct {
		Remove []string
		Rename map[string]string
	}]("base-type-changes.json")
	selections := []any{}
	obsolete := map[string]bool{}
	for _, name := range changes.Remove {
		selections = append(selections, name)
		obsolete[name] = true
	}
	for old := range changes.Rename {
		selections = append(selections, old)
		obsolete[old] = true
	}
	for _, name := range changes.Rename {
		delete(obsolete, name)
	}
	mustCall(t, s, "saveDefaults", Object{"defaults": Object{"currency": Object{
		"assignments": Object{"Coin of Skill": "s"},
		"items":       Object{"Coin of Skill": Object{"visibility": "Hide"}},
	}}})
	const name = "Previous League"
	mustCall(t, s, "createFilter", Object{"name": name})
	cfg := Object{
		"buildProfile": Object{"weaponPhases": []any{Object{"baseTypes": selections}}},
		"buildSpecificOptions": Object{
			"highlightedEquipment": Object{"highlights": []any{Object{"baseTypes": selections}}},
			"gemCallouts":          Object{"styleOnly": []any{"Her Mask", "Maven's Chisel of Scarabs"}},
			"currency":             Object{"items": Object{"Coin of Skill": Object{"visibility": "Show", "sound": Object{"soundId": float64(7)}}}},
		},
		"localDefaults": Object{"tinctures": Object{"baseTypes": []any{"Orb of Binding"}}},
	}
	mustCall(t, s, "saveConfig", Object{"name": name, "config": cfg})
	path, err := s.filterPath(name)
	if err != nil {
		t.Fatal(err)
	}
	readSaved := func(path string) Object {
		t.Helper()
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var saved Object
		if err := json.Unmarshal(b, &saved); err != nil {
			t.Fatal(err)
		}
		return saved
	}
	before := readSaved(path)
	defaultsPath := filepath.Join(s.Root, "defaults.json")
	beforeDefaults := readSaved(defaultsPath)
	check := func(output string) {
		t.Helper()
		for name := range obsolete {
			if strings.Contains(output, strconv.Quote(name)) {
				t.Error("obsolete name reached preview/export", name)
			}
		}
		last := strings.Split(output, "\n\n")
		if !strings.Contains(output, `"Maven's Chisel of Threads"`) || len(last) < 2 || last[len(last)-2] != "### Show Unknown Items" || !strings.HasPrefix(last[len(last)-1], "Show\n") {
			t.Fatal("renamed selection or final fallback lost")
		}
		found := false
		currencySection := strings.SplitN(strings.SplitN(output, "### Currency", 2)[1], "### Scrolls", 2)[0]
		for _, block := range strings.Split(currencySection, "\n\n") {
			if strings.Contains(block, `BaseType == "Soul Core of Finesse"`) {
				found = true
				if !strings.HasPrefix(block, "Show\n") || !strings.Contains(block, "PlayAlertSound 7 240") {
					t.Fatal("old filter currency choices were lost", block)
				}
			}
		}
		if !found {
			t.Fatal("renamed currency rule missing")
		}
	}
	preview := c.Text(mustCall(t, s, "preview", Object{"name": name}).(Object)["output"])
	check(preview)
	if !reflect.DeepEqual(before, readSaved(path)) || !reflect.DeepEqual(beforeDefaults, readSaved(defaultsPath)) {
		t.Fatal("preview rewrote saved item names or defaults")
	}
	mustCall(t, s, "exportFilter", Object{"name": name})
	exported, err := os.ReadFile(filepath.Join(game, name+".filter"))
	if err != nil {
		t.Fatal(err)
	}
	check(string(exported))
	if string(exported) != preview {
		t.Fatal("preview and single export differ")
	}
	if !reflect.DeepEqual(c.CurrentItemConfig(before), readSaved(path)) || !reflect.DeepEqual(c.CurrentItemDefaults(beforeDefaults), readSaved(defaultsPath)) {
		t.Fatal("single export did not save cleaned names")
	}
	// Exercise Export All on historical JSON again, rather than already-cleaned data.
	mustCall(t, s, "saveConfig", Object{"name": name, "config": before})
	mustCall(t, s, "saveDefaults", Object{"defaults": beforeDefaults})
	all := mustCall(t, s, "exportAllFilters", nil).(Object)
	if len(all["errors"].([]Object)) != 0 || all["exported"] != 3 {
		t.Fatal("Export All failed", all)
	}
	files, err := filepath.Glob(filepath.Join(game, "*.filter"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for name := range obsolete {
			if strings.Contains(string(b), strconv.Quote(name)) {
				t.Error("Export All retained obsolete name", file, name)
			}
		}
	}
	if !reflect.DeepEqual(c.CurrentItemConfig(before), readSaved(path)) || !reflect.DeepEqual(c.CurrentItemDefaults(beforeDefaults), readSaved(defaultsPath)) {
		t.Fatal("re-export did not persist current item names and defaults")
	}
	if !reflect.DeepEqual(readSaved(path), mustCall(t, s, "getConfig", Object{"name": name})) {
		t.Fatal("editor reload did not return the saved current names")
	}
}

func TestFailedExportDoesNotPersistItemCleanup(t *testing.T) {
	for _, failure := range []string{"sounds", "game file"} {
		t.Run(failure, func(t *testing.T) {
			s := testStore(t)
			game := t.TempDir()
			settings, _ := s.settings()
			settings["filterPath"] = game
			mustCall(t, s, "saveSettings", Object{"settings": settings})
			cfg := Object{
				"buildProfile":         Object{"weaponPhases": []any{Object{"baseTypes": []any{"Orb of Binding"}}}},
				"buildSpecificOptions": Object{},
			}
			defaults := Object{"currency": Object{"assignments": Object{"Coin of Skill": "s"}}}
			mustCall(t, s, "saveConfig", Object{"name": "Caster Template", "config": cfg})
			mustCall(t, s, "saveDefaults", Object{"defaults": defaults})
			s.FetchAudio = func(string, string) ([]byte, error) { return nil, os.ErrNotExist }
			if failure == "game file" {
				s.FetchAudio = func(string, string) ([]byte, error) { return []byte("test"), nil }
				s.AudioProcessor = func(b []byte, path string, _ float64) error { return os.WriteFile(path, b, 0600) }
				if err := os.Mkdir(filepath.Join(game, "Caster Template.filter"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			path, _ := s.filterPath("Caster Template")
			before, _ := os.ReadFile(path)
			defaultsPath := filepath.Join(s.Root, "defaults.json")
			beforeDefaults, _ := os.ReadFile(defaultsPath)
			if _, err := s.Dispatch("exportFilter", Object{"name": "Caster Template", "config": cfg}); err == nil {
				t.Fatal("expected export error")
			}
			after, _ := os.ReadFile(path)
			afterDefaults, _ := os.ReadFile(defaultsPath)
			if string(before) != string(after) || string(beforeDefaults) != string(afterDefaults) {
				t.Fatal("failed export rewrote user data")
			}
		})
	}
}

func TestExportReportsJSONSaveFailureAfterGameWrite(t *testing.T) {
	s := testStore(t)
	settings, _ := s.settings()
	game := t.TempDir()
	settings["filterPath"] = game
	mustCall(t, s, "saveSettings", Object{"settings": settings})
	const name = "Caster Template"
	cfg := Object{"buildProfile": Object{}, "buildSpecificOptions": Object{"gemCallouts": Object{"styleOnly": []any{"Coin of Skill"}}}}
	mustCall(t, s, "saveConfig", Object{"name": name, "config": cfg})
	path, _ := s.filterPath(name)
	before, _ := os.ReadFile(path)
	s.FetchAudio = func(string, string) ([]byte, error) { return []byte("test"), nil }
	blocked := false
	s.AudioProcessor = func(b []byte, soundPath string, _ float64) error {
		if !blocked {
			blocked = true
			// Simulate losing permission to replace the JSON after compilation,
			// without depending on chmod (tests can run as root or on Windows).
			if err := os.Rename(path, path+".previous"); err != nil {
				return err
			}
			if err := os.Mkdir(path, 0700); err != nil {
				return err
			}
		}
		return os.WriteFile(soundPath, b, 0600)
	}
	_, err := s.Dispatch("exportFilter", Object{"name": name})
	if err == nil || !strings.Contains(err.Error(), "exported Caster Template.filter, but could not save updated filter JSON") {
		t.Fatal("missing explicit partial-success error", err)
	}
	output, err := os.ReadFile(filepath.Join(game, exportName(name)))
	if err != nil || !strings.Contains(string(output), `"Soul Core of Finesse"`) {
		t.Fatal("expected successful current-name game export", err)
	}
	after, _ := os.ReadFile(path + ".previous")
	if string(before) != string(after) {
		t.Fatal("previous JSON was modified")
	}
}
