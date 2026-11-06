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

func TestLocalCurrencySoundReplacesGlobalSource(t *testing.T) {
	s := testStore(t)
	mustCall(t, s, "saveDefaults", Object{"defaults": Object{"currency": Object{"items": Object{
		"Chaos Orb": Object{"sound": Object{"enabled": true, "tts": "chaos_orb"}},
	}}}})
	cfg := Object{"buildProfile": Object{}, "buildSpecificOptions": Object{}, "localDefaults": Object{"currency": Object{"items": Object{
		"Chaos Orb": Object{"sound": Object{"enabled": true, "soundId": float64(8)}},
	}}}}
	mustCall(t, s, "saveConfig", Object{"name": "Caster Template", "config": cfg})
	defaults := c.Obj(mustCall(t, s, "filterDefaults", Object{"name": "Caster Template"}).(Object)["defaults"])
	sound := c.Obj(c.Obj(c.Obj(c.Obj(defaults["currency"])["items"])["Chaos Orb"])["sound"])
	if sound["tts"] != nil || sound["soundId"] != float64(8) {
		t.Fatal("local sound retained the global source", sound)
	}
	out := mustCall(t, s, "preview", Object{"name": "Caster Template"}).(Object)["output"].(string)
	found := false
	for _, block := range strings.Split(out, "\n\n") {
		if !strings.Contains(block, `BaseType == "Chaos Orb"`) {
			continue
		}
		found = true
		if !strings.Contains(block, "PlayAlertSound 8 240") || strings.Contains(block, "CustomAlertSound") {
			t.Fatal("compiler did not use the local sound", block)
		}
	}
	if !found {
		t.Fatal("missing Chaos Orb rule")
	}
	shared := mustCall(t, s, "share", Object{"name": "Caster Template", "includeGlobals": true}).(Object)
	var snapshot Object
	if err := json.Unmarshal([]byte(c.Text(shared["text"])), &snapshot); err != nil {
		t.Fatal(err)
	}
	sharedSound := c.Obj(c.Obj(c.Obj(c.Obj(c.Obj(snapshot["defaults"])["currency"])["items"])["Chaos Orb"])["sound"])
	if sharedSound["tts"] != nil || sharedSound["soundId"] != float64(8) {
		t.Fatal("sharing retained the global sound source", sharedSound)
	}
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

func TestReferenceIncludesNewPresentationPresets(t *testing.T) {
	s := testStore(t)
	ref := mustCall(t, s, "reference", Object{}).(Object)
	names := ref["styleNames"].([]string)
	if c.Obj(c.Obj(ref["currencySounds"])["Chromatic Orb"])["tts"] != "chromatic_orb" {
		t.Fatal("reference must include Chromatic Orb's actual default sound")
	}
	if icon := c.Obj(c.Obj(ref["currencyIcons"])["Chromatic Orb"]); icon["iconShape"] != "Pentagon" || icon["iconColor"] != "Purple" || icon["iconSize"] != float64(2) {
		t.Fatal("reference must include Chromatic Orb's actual default icon", icon)
	}
	for _, name := range []string{"currencyLow", "currencyLeaguestart", "leagueMechanic"} {
		if !contains(names, name) || len(c.Obj(c.Obj(ref["styles"])[name])) == 0 {
			t.Fatalf("missing selectable preset %s", name)
		}
	}
}

func TestReferenceSeparatesGlobalAndLocalStyles(t *testing.T) {
	s := testStore(t)
	mustCall(t, s, "createFilter", Object{"name": "Local Styles"})
	mustCall(t, s, "saveStyles", Object{"styles": Object{"gemCallout": Object{"text": "#123456"}}})
	mustCall(t, s, "saveConfig", Object{"name": "Local Styles", "config": Object{
		"buildProfile": Object{}, "buildSpecificOptions": Object{},
		"localStyles": Object{"gemCallout": Object{"text": "#abcdef"}},
	}})
	ref := mustCall(t, s, "reference", Object{"name": "Local Styles"}).(Object)
	if c.Obj(c.Obj(ref["styles"])["gemHighlight"])["text"] != "#abcdef" {
		t.Fatal("effective reference must include local styles")
	}
	if c.Obj(c.Obj(ref["globalStyles"])["gemHighlight"])["text"] != "#123456" {
		t.Fatal("reset baseline must exclude local styles")
	}
	if c.Obj(c.Obj(ref["styles"])["gemHighlight"])["text"] != "#abcdef" {
		t.Fatal("legacy local gem override must reach the renamed style")
	}
	for _, name := range ref["styleNames"].([]string) {
		if c.LegacyStyle(name) {
			t.Fatal("retired style offered for new selections", name)
		}
	}
}

func TestGlobalCurrencyAndLeagueDefaultsReachFilters(t *testing.T) {
	s := testStore(t)
	mustCall(t, s, "createFilter", Object{"name": "Shared Currency"})
	mustCall(t, s, "saveDefaults", Object{"defaults": Object{
		"currency":       Object{"assignments": Object{"Chromatic Orb": "leaguestart"}, "items": Object{"Chromatic Orb": Object{"sound": Object{"soundId": float64(8)}}}},
		"leagueMechanic": Object{"rules": Object{"charts": Object{"baseTypes": []any{}}}},
	}})
	mustCall(t, s, "saveStyles", Object{"styles": Object{"currencyLeaguestart": Object{"text": "#123456"}}})
	blockFor := func(output, name string) string {
		for _, block := range strings.Split(output, "\n\n") {
			if strings.Contains(block, "BaseType == ") && strings.Contains(block, `"`+name+`"`) {
				return block
			}
		}
		return ""
	}
	output := c.Text(mustCall(t, s, "preview", Object{"name": "Shared Currency"}).(Object)["output"])
	chromatic := blockFor(output, "Chromatic Orb")
	if !strings.Contains(chromatic, "PlayAlertSound 8 240") || !strings.Contains(chromatic, "SetTextColor 18 52 86") {
		t.Fatal("global currency defaults/styles not applied to filter")
	}
	if !strings.HasPrefix(blockFor(output, "Sandy Seabed Chart"), "Hide\n") {
		t.Fatal("global empty league selection must hide charts")
	}
	config := Object{"buildProfile": Object{}, "buildSpecificOptions": Object{
		"currency":       Object{"items": Object{"Chromatic Orb": Object{"visibility": "Hide"}}},
		"leagueMechanic": Object{"rules": Object{"charts": Object{"baseTypes": []any{"Sandy Seabed Chart"}}}},
	}}
	output = c.Text(mustCall(t, s, "preview", Object{"name": "Shared Currency", "config": config}).(Object)["output"])
	if !strings.HasPrefix(blockFor(output, "Chromatic Orb"), "Hide\n") || !strings.HasPrefix(blockFor(output, "Sandy Seabed Chart"), "Show\n") {
		t.Fatal("filter-specific settings must override global item defaults")
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

func TestRemovedRuleOverridesAreDiscardedOnLoadAndSave(t *testing.T) {
	s := testStore(t)
	mustCall(t, s, "createFilter", Object{"name": "Simple Rules"})
	config := Object{"buildProfile": Object{}, "buildSpecificOptions": Object{
		"early":                Object{"earlyMaxAreaLevel": float64(24), "rules": Object{"armour2": Object{"maxAreaLevel": float64(2), "style": Object{"size": float64(22)}}}},
		"currency":             Object{"items": Object{"Chaos Orb": Object{"visibility": "Hide"}}},
		"highlightedEquipment": Object{"highlights": []any{Object{"baseTypes": []any{"Iron Ring"}, "style": Object{"text": "#123456"}}}},
	}, "localDefaults": Object{"links": Object{"rules": Object{"3generic": Object{"size": float64(20)}}}}}
	path, err := s.filterPath("Simple Rules")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(path, config); err != nil {
		t.Fatal(err)
	}
	loaded := mustCall(t, s, "getConfig", Object{"name": "Simple Rules"}).(Object)
	if _, exists := c.Obj(c.Obj(loaded["buildSpecificOptions"])["early"])["rules"]; exists {
		t.Fatal("obsolete filter rules were loaded")
	}
	if _, exists := c.Obj(c.Obj(loaded["localDefaults"])["links"])["rules"]; exists {
		t.Fatal("filter-local rules were loaded")
	}
	mustCall(t, s, "saveConfig", Object{"name": "Simple Rules", "config": config})
	saved, err := readObject(path)
	if err != nil {
		t.Fatal(err)
	}
	options := c.Obj(saved["buildSpecificOptions"])
	if _, exists := c.Obj(options["early"])["rules"]; exists {
		t.Fatal("obsolete filter rules were saved")
	}
	if c.Obj(options["early"])["earlyMaxAreaLevel"] != float64(24) || c.Obj(c.Obj(c.Obj(options["currency"])["items"])["Chaos Orb"])["visibility"] != "Hide" || len(c.List(c.Obj(options["highlightedEquipment"])["highlights"])) != 1 {
		t.Fatal("unrelated overrides were lost")
	}
}

func TestGlobalRulesSaveOnlyMaximumAndSizeAndApplyToFilters(t *testing.T) {
	s := testStore(t)
	mustCall(t, s, "createFilter", Object{"name": "Global Rule Tuning"})
	mustCall(t, s, "saveDefaults", Object{"defaults": Object{"early": Object{"rules": Object{"armour2": Object{
		"maxAreaLevel": float64(17), "size": float64(31), "enabled": false,
		"style": Object{"text": "#123456", "size": float64(22)}, "icon": Object{"iconShape": "Star"}, "sound": Object{"soundId": float64(8)},
	}}}}})
	saved, err := s.optional("defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	rule := c.Obj(c.Obj(c.Obj(saved["early"])["rules"])["armour2"])
	if len(rule) != 2 || rule["size"] != float64(31) || rule["maxAreaLevel"] != float64(17) {
		t.Fatal("obsolete rule overrides retained or migrated", rule)
	}
	out := c.Text(mustCall(t, s, "preview", Object{"name": "Global Rule Tuning"}).(Object)["output"])
	start := strings.Index(out, "### Early Sockets")
	if start < 0 {
		t.Fatal("missing Early Sockets output")
	}
	end := strings.Index(out[start:], "### Flasks")
	if end < 0 {
		t.Fatal("missing Flasks output")
	}
	section := out[start : start+end]
	if !strings.Contains(section, "AreaLevel <= 17") || !strings.Contains(section, "SetFontSize 31") || strings.Contains(section, "Hide\n") || strings.Contains(section, "AlertSound") {
		t.Fatal("global numeric tuning not applied cleanly", section)
	}
}

func TestGlobalFlaskCutoffsAndStylesReachFilterPreviews(t *testing.T) {
	s := testStore(t)
	mustCall(t, s, "createFilter", Object{"name": "Flask Cutoffs"})
	mustCall(t, s, "saveDefaults", Object{"defaults": Object{"flasks": Object{
		"lifeMaxAreaLevels": Object{"small": float64(18)},
		"manaMaxAreaLevels": Object{"eternal": float64(72)},
	}}})
	mustCall(t, s, "saveStyles", Object{"styles": Object{"sharedRarityNormal": Object{"text": "#123456"}}})
	blockFor := func(output, base string) string {
		for _, block := range strings.Split(output, "\n\n") {
			if strings.Contains(block, `BaseType == "`+base+`"`) {
				return block
			}
		}
		return ""
	}
	out := c.Text(mustCall(t, s, "preview", Object{"name": "Flask Cutoffs"}).(Object)["output"])
	small := blockFor(out, "Small Life Flask")
	if !strings.Contains(small, "AreaLevel <= 18") || !strings.Contains(small, "SetTextColor 18 52 86") || !strings.Contains(small, "life.mp3") {
		t.Fatal("global flask cutoffs, styles or automatic TTS missing", small)
	}
	if !strings.Contains(blockFor(out, "Eternal Mana Flask"), "AreaLevel <= 72") {
		t.Fatal("global final-flask cutoff missing")
	}
	config := Object{"buildProfile": Object{}, "buildSpecificOptions": Object{"flasks": Object{"lifeMaxAreaLevels": Object{"small": nil}}}, "localStyles": Object{"sharedRarityNormal": Object{"text": "#654321"}}}
	mustCall(t, s, "saveConfig", Object{"name": "Flask Cutoffs", "config": config})
	out = c.Text(mustCall(t, s, "preview", Object{"name": "Flask Cutoffs"}).(Object)["output"])
	if strings.Contains(blockFor(out, "Small Life Flask"), "AreaLevel") {
		t.Fatal("saved explicit null must remove inherited cutoff")
	}
	if !strings.Contains(blockFor(out, "Small Life Flask"), "SetTextColor 101 67 33") {
		t.Fatal("saved local palette must override shared global text")
	}
	if !strings.Contains(blockFor(out, "Eternal Mana Flask"), "AreaLevel <= 72") {
		t.Fatal("filter override lost unrelated global cutoff")
	}
}

func TestGenerateSoundsAlsoCleansCacheAndSyncs(t *testing.T) {
	s := testStore(t)
	game := t.TempDir()
	settings, _ := s.settings()
	settings["filterPath"] = game
	mustCall(t, s, "saveSettings", Object{"settings": settings})
	s.FetchAudio = func(string, string) ([]byte, error) { return []byte("fake TTS"), nil }
	s.AudioProcessor = func(b []byte, path string, _ float64) error { return os.WriteFile(path, b, 0600) }
	for _, name := range []string{"old-generated.mp3", "personal.mp3"} {
		if err := os.WriteFile(filepath.Join(s.Root, "sounds", name), []byte("audio"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.write("sound-state.json", Object{"old-generated.mp3": "old-signature"}); err != nil {
		t.Fatal(err)
	}
	mustCall(t, s, "generateSounds", nil)
	if _, err := os.Stat(filepath.Join(s.Root, "sounds", "old-generated.mp3")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("stale generated cache remains", err)
	}
	if _, err := os.Stat(filepath.Join(game, soundFolder(settings), "old-generated.mp3")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("stale generated cache was copied to PoE", err)
	}
	for _, name := range []string{"chaos_orb.mp3", "personal.mp3"} {
		if _, err := os.Stat(filepath.Join(game, soundFolder(settings), name)); err != nil {
			t.Fatal("sound not synced", name, err)
		}
	}
	// Unchanged generated sounds must still work offline.
	s.FetchAudio = func(string, string) ([]byte, error) { return nil, errors.New("offline") }
	mustCall(t, s, "generateSounds", nil)
}

func TestAudioStatusReportsMissingFFmpeg(t *testing.T) {
	s := testStore(t)
	result := mustCall(t, s, "audioStatus", nil).(Object)
	if result["ffmpegAvailable"] != false {
		t.Fatal("FFmpeg should be reported unavailable when no processor is configured", result)
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

func TestExportPreservesSpacesAndCaseInFilterName(t *testing.T) {
	s := testStore(t)
	game := t.TempDir()
	settings, _ := s.settings()
	settings["filterPath"] = game
	mustCall(t, s, "saveSettings", Object{"settings": settings})
	s.FetchAudio = func(string, string) ([]byte, error) { return []byte("fake TTS"), nil }
	s.AudioProcessor = func(b []byte, path string, _ float64) error { return os.WriteFile(path, b, 0600) }
	mustCall(t, s, "createFilter", Object{"name": "test abc"})
	// Existing game files must not be overwritten or removed through the old
	// name normalization, even when they differ only by spaces and casing.
	otherPath := filepath.Join(game, "TestAbc.filter")
	if err := os.WriteFile(otherPath, []byte("unrelated filter"), 0600); err != nil {
		t.Fatal(err)
	}
	result := mustCall(t, s, "exportFilter", Object{"name": "test abc"}).(Object)
	if result["fileName"] != "test abc.filter" {
		t.Fatal("export changed the filter name", result)
	}
	path := filepath.Join(game, "test abc.filter")
	if _, err := os.Stat(path); err != nil {
		t.Fatal("missing export with the original name", err)
	}
	mustCall(t, s, "deleteFilter", Object{"name": "test abc", "deleteGameFile": true})
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("delete left the named export behind", err)
	}
	other, err := os.ReadFile(otherPath)
	if err != nil || string(other) != "unrelated filter" {
		t.Fatal("export or deletion touched another game file", err)
	}
}
