package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	c "github.com/mositho/filtertool-desktop/internal/compiler"
)

func TestPreserveJSONOrderOnSourceSave(t *testing.T) {
	original := []byte(`{"zeta":{"last":1,"first":2},"alpha":3}`)
	prior, err := decodeOrderedJSON(original)
	if err != nil {
		t.Fatal(err)
	}
	updated := preserveJSONOrder(Object{"alpha": float64(4), "zeta": Object{"first": float64(2), "last": float64(9)}}, prior)
	got, err := json.Marshal(updated)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"zeta":{"last":9,"first":2},"alpha":4}` {
		t.Fatalf("save changed JSON key order or values: %s", got)
	}
}

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Caster Template", "Attack Template"} {
		mustCall(t, s, "createFilter", Object{"name": name, "template": name})
	}
	return s
}
func mustCall(t *testing.T, s *Store, action string, input Object) any {
	t.Helper()
	v, err := s.Dispatch(action, input)
	if err != nil {
		t.Fatalf("%s: %v", action, err)
	}
	return v
}
func TestPersonalLaunchDoesNotSeedTemplates(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	filters, err := s.filters()
	if err != nil || len(filters) != 0 {
		t.Fatal(filters, err)
	}
	mustCall(t, s, "createFilter", Object{"name": "My Caster", "template": "Caster Template"})
	if _, err = Open(s.Root); err != nil {
		t.Fatal(err)
	}
	filters, _ = s.filters()
	if len(filters) != 1 || filters[0]["name"] != "My Caster" {
		t.Fatal("personal filters changed on reopening", filters)
	}
}
func TestLifecycleValidationAndPreview(t *testing.T) {
	s := testStore(t)
	mustCall(t, s, "createFilter", Object{"name": "My Filter"})
	for _, name := range []string{"../outside", "CON", "My-Filter", "my filter", " trailing "} {
		if _, err := s.Dispatch("createFilter", Object{"name": name}); err == nil {
			t.Errorf("accepted unsafe/colliding name %q", name)
		}
	}
	cfg := Object{"buildProfile": Object{"preferredColors": []any{}}, "buildSpecificOptions": Object{}, "localStyles": Object{"currencyA": Object{"text": "#abc"}}}
	mustCall(t, s, "saveConfig", Object{"name": "My Filter", "config": cfg})
	before, _ := os.ReadFile(filepath.Join(s.Root, "filters", "My Filter.json"))
	out := mustCall(t, s, "preview", Object{"name": "My Filter"}).(Object)
	after, _ := os.ReadFile(filepath.Join(s.Root, "filters", "My Filter.json"))
	if string(before) != string(after) {
		t.Fatal("preview modified filter")
	}
	if !strings.Contains(c.Text(out["output"]), "SetTextColor 170 187 204") {
		t.Fatal("local styles ignored")
	}
	if _, err := os.Stat(filepath.Join(s.Root, "sound-state.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("preview generated sounds")
	}
	mustCall(t, s, "duplicateFilter", Object{"name": "My Filter", "newName": "Duplicate"})
	mustCall(t, s, "renameFilter", Object{"name": "Duplicate", "newName": "Renamed"})
	if _, err := s.readFilter("Duplicate"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("old filter remains")
	}
	mustCall(t, s, "deleteFilter", Object{"name": "Renamed"})
}
func TestSharePreservesOutputWithoutChangingRecipientGlobals(t *testing.T) {
	sender, recipient := testStore(t), testStore(t)
	mustCall(t, sender, "saveDefaults", Object{"defaults": Object{"preferredColors": []any{"B"}, "links": Object{"fourLinkMaxAreaLevel": float64(40)}}})
	mustCall(t, sender, "saveStyles", Object{"styles": Object{"currencyA": Object{"text": "#123456"}}})
	mustCall(t, recipient, "saveDefaults", Object{"defaults": Object{"preferredColors": []any{"R"}, "rareItems": Object{"maxAreaLevel": float64(20)}}})
	mustCall(t, recipient, "saveStyles", Object{"styles": Object{"currencyA": Object{"size": float64(25), "backgroundOpacity": 0.2}}})
	beforeD, _ := os.ReadFile(filepath.Join(recipient.Root, "defaults.json"))
	beforeS, _ := os.ReadFile(filepath.Join(recipient.Root, "styles.json"))
	shared := mustCall(t, sender, "share", Object{"name": "Caster Template", "includeGlobals": true}).(Object)
	text := c.Text(shared["text"])
	if strings.ContainsAny(text, "\n\r") {
		t.Fatal("share was not minified")
	}
	var value Object
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		t.Fatal(err)
	}
	mustCall(t, recipient, "importJSON", Object{"name": "Imported", "value": value})
	want := mustCall(t, sender, "preview", Object{"name": "Caster Template"}).(Object)["output"]
	got := mustCall(t, recipient, "preview", Object{"name": "Imported"}).(Object)["output"]
	if got != want {
		t.Fatal("shared filter changed behavior on recipient")
	}
	afterD, _ := os.ReadFile(filepath.Join(recipient.Root, "defaults.json"))
	afterS, _ := os.ReadFile(filepath.Join(recipient.Root, "styles.json"))
	if string(beforeD) != string(afterD) || string(beforeS) != string(afterS) {
		t.Fatal("import modified global settings")
	}
	if len(c.Obj(c.Obj(mustCall(t, recipient, "getConfig", Object{"name": "Imported"}))["localStyles"])) == 0 {
		t.Fatal("local style differences missing")
	}
	raw := mustCall(t, sender, "share", Object{"name": "Caster Template", "includeGlobals": false}).(Object)
	if strings.Contains(c.Text(raw["text"]), `"format"`) {
		t.Fatal("raw share contains global envelope")
	}
}
func TestWebJSONAndFolderImport(t *testing.T) {
	s := testStore(t)
	root := t.TempDir()
	dir := filepath.Join(root, "src", "filters", "Old Filter")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	source := []byte(`{"buildProfile":{"preferredColors":[]},"buildSpecificOptions":{}}`)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.ts"), []byte("throw new Error('must never execute')"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := s.importFolder(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result["imported"].([]string)) != 1 {
		t.Fatal(result)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	if string(after) != string(source) {
		t.Fatal("migration modified source")
	}
	cfg, err := s.readFilter("Old Filter")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.List(c.Obj(cfg["buildProfile"])["preferredColors"])) != 0 {
		t.Fatal("lost explicit empty array")
	}
	if _, err = s.importJSON("Invalid", Object{"hello": "world"}, nil, nil); err == nil {
		t.Fatal("accepted unrelated JSON")
	}
	if _, err = s.importJSON("Empty Legacy", Object{}, nil, nil); err != nil {
		t.Fatal("empty legacy share must remain importable", err)
	}
}

func TestSoundSyncOnlyDeletesUnusedGeneratedCache(t *testing.T) {
	s := testStore(t)
	settings, _ := s.settings()
	settings["filterPath"] = t.TempDir()
	mustCall(t, s, "saveSettings", Object{"settings": settings})
	for _, name := range []string{"old-generated.mp3", "personal.mp3"} {
		if err := os.WriteFile(filepath.Join(s.Root, "sounds", name), []byte("audio"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.write("sound-state.json", Object{"old-generated.mp3": "old-signature"}); err != nil {
		t.Fatal(err)
	}
	result := mustCall(t, s, "syncSounds", nil).(Object)
	if result["removed"] != 1 {
		t.Fatal(result)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "sounds", "personal.mp3")); err != nil {
		t.Fatal("custom audio removed", err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "sounds", "old-generated.mp3")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("stale generated audio remains")
	}
}
func TestExportOfflineAndNoPartialGameWrite(t *testing.T) {
	s := testStore(t)
	game := t.TempDir()
	settings, _ := s.settings()
	settings["filterPath"] = game
	mustCall(t, s, "saveSettings", Object{"settings": settings})
	s.FetchAudio = func(string, string) ([]byte, error) { return []byte("fake TTS"), nil }
	s.AudioProcessor = func(b []byte, path string, _ float64) error { return os.WriteFile(path, b, 0600) }
	result := mustCall(t, s, "exportFilter", Object{"name": "Caster Template"}).(Object)
	path := filepath.Join(game, c.Text(result["fileName"]))
	out, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(out), "### Show Unknown Items") {
		t.Fatal("missing export", err)
	}
	if _, err = os.Stat(filepath.Join(game, "poeft-sounds-v2", "chaos_orb.mp3")); err != nil {
		t.Fatal("missing sound pack", err)
	}
	settings["tts"] = Object{"locale": "en-US", "speed": float64(1.7)}
	mustCall(t, s, "saveSettings", Object{"settings": settings})
	s.FetchAudio = func(string, string) ([]byte, error) { return nil, errors.New("offline") }
	if _, err = s.Dispatch("exportFilter", Object{"name": "Caster Template"}); err == nil {
		t.Fatal("expected sound-generation error")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(out) {
		t.Fatal("failed export overwrote working game filter")
	}
}
