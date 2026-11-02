package app

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	c "github.com/mositho/filtertool-desktop/internal/compiler"
	"github.com/mositho/filtertool-desktop/internal/data"
)

func TestBundledTemplateSelections(t *testing.T) {
	templates := data.Load[Object]("templates.json")
	caster := c.Obj(templates["Caster Template"])
	attack := c.Obj(templates["Attack Template"])
	profile := c.Obj(caster["buildProfile"])
	if !reflect.DeepEqual(c.Strings(profile["preferredArmour"]), []string{"es", "es-evasion"}) || profile["shieldProgression"] != "full" || c.Obj(profile["misc"])["whetstoneRecipe"] != false {
		t.Fatal("incorrect caster profile", profile)
	}
	if !reflect.DeepEqual(c.Strings(c.Obj(c.Obj(caster["buildSpecificOptions"])["jewellery"])["amulets"]), []string{"Amber", "Jade", "Turquoise", "Citrine", "Agate", "Onyx"}) {
		t.Fatal("incorrect caster amulets")
	}
	if c.Obj(attack["buildProfile"])["shieldProgression"] != "early" || !reflect.DeepEqual(c.Strings(c.Obj(c.Obj(attack["buildSpecificOptions"])["jewellery"])["amulets"]), []string{"Jade", "Lapis", "Turquoise", "Citrine", "Agate", "Onyx"}) {
		t.Fatal("incorrect attack selections")
	}
}

func devSource(t *testing.T) string {
	t.Helper()
	source := t.TempDir()
	for _, name := range []string{"templates.json", "defaults.json", "styles.json"} {
		contents, err := data.Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(source, name), contents, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return source
}

func TestDevWorkspaceCanCreateIsolatedTestFilters(t *testing.T) {
	root, source := t.TempDir(), devSource(t)
	before, err := os.ReadFile(filepath.Join(source, "templates.json"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := OpenDev(root, source)
	if err != nil {
		t.Fatal(err)
	}
	mustCall(t, s, "createFilter", Object{"name": "Test Filter"})
	cfg := mustCall(t, s, "getConfig", Object{"name": "Test Filter"}).(Object)
	if len(c.Obj(cfg["buildProfile"])) != 0 || len(c.Obj(cfg["buildSpecificOptions"])) != 0 {
		t.Fatal("new test filter is not blank", cfg)
	}
	c.Obj(cfg["buildProfile"])["preferredColors"] = []any{"R"}
	mustCall(t, s, "saveConfig", Object{"name": "Test Filter", "config": cfg})
	list := mustCall(t, s, "listFilters", nil).(Object)["filters"].([]Object)
	if len(list) != 3 {
		t.Fatal("test filters missing from dev list", list)
	}
	found := false
	for _, filter := range list {
		if filter["name"] == "Test Filter" {
			found = filter["devTest"] == true
		}
	}
	if !found {
		t.Fatal("test filter must be identified separately from templates")
	}
	reopened, err := OpenDev(root, source)
	if err != nil {
		t.Fatal(err)
	}
	loaded := mustCall(t, reopened, "getConfig", Object{"name": "Test Filter"}).(Object)
	if !reflect.DeepEqual(c.Strings(c.Obj(loaded["buildProfile"])["preferredColors"]), []string{"R"}) {
		t.Fatal("test filter did not survive restart")
	}
	after, _ := os.ReadFile(filepath.Join(source, "templates.json"))
	if string(after) != string(before) {
		t.Fatal("creating/saving test filter changed repository templates")
	}
	if _, err := os.Stat(filepath.Join(root, "filters", "Test Filter.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("test filter touched personal profile")
	}
	if _, err := s.Dispatch("createFilter", Object{"name": "Caster Template"}); err == nil {
		t.Fatal("test filter creation replaced a bundled template")
	}
	if _, err := s.Dispatch("exportFilter", Object{"name": "Test Filter"}); err == nil {
		t.Fatal("dev test filter export must stay disabled")
	}
}

func TestDevWorkspaceAddsAndDeletesTemplatesAndTestFilters(t *testing.T) {
	s, err := OpenDev(t.TempDir(), devSource(t))
	if err != nil {
		t.Fatal(err)
	}
	mustCall(t, s, "createFilter", Object{"name": "Leveling Template"})
	mustCall(t, s, "createFilter", Object{"name": "Temporary Run"})
	source, err := s.source("templates")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := source["Leveling Template"]; !ok {
		t.Fatal("new template was not added to repository source")
	}
	if _, ok := source["Temporary Run"]; ok {
		t.Fatal("test filter leaked into repository templates")
	}
	// Even an old non-template source entry must not appear as a starting template.
	source["Scratch"] = Object{"buildProfile": Object{}, "buildSpecificOptions": Object{}}
	if err := s.writeSource("templates", source); err != nil {
		t.Fatal(err)
	}
	list := mustCall(t, s, "listFilters", nil).(Object)
	choices := list["templates"].([]string)
	if len(choices) != 3 || !contains(choices, "Leveling Template") || contains(choices, "Scratch") || contains(choices, "Temporary Run") {
		t.Fatal("incorrect starting-template choices", choices)
	}
	if _, err := s.Dispatch("createFilter", Object{"name": "From Scratch", "template": "Scratch"}); err == nil {
		t.Fatal("non-template accepted as starting template")
	}
	mustCall(t, s, "createFilter", Object{"name": "From Template", "template": "Leveling Template"})
	// A previously hidden profile copy must not resurrect a deleted template.
	copyPath, _ := s.filterPath("Leveling Template")
	if err := writeJSON(copyPath, Object{"buildProfile": Object{}, "buildSpecificOptions": Object{}}); err != nil {
		t.Fatal(err)
	}
	game := t.TempDir()
	gameFile := filepath.Join(game, exportName("Temporary Run"))
	if err := os.WriteFile(gameFile, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	settings, _ := s.settings()
	settings["filterPath"] = game
	mustCall(t, s, "saveSettings", Object{"settings": settings})
	for _, name := range []string{"Leveling Template", "Temporary Run", "Scratch", "From Template"} {
		mustCall(t, s, "deleteFilter", Object{"name": name, "deleteGameFile": true})
		if _, err := s.Dispatch("getConfig", Object{"name": name}); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("deleted entry is still accessible", name, err)
		}
	}
	if _, err := os.Stat(gameFile); err != nil {
		t.Fatal("dev deletion touched game exports", err)
	}
	list = mustCall(t, s, "listFilters", nil).(Object)
	if len(list["filters"].([]Object)) != 2 || len(list["templates"].([]string)) != 2 {
		t.Fatal("deleted entries remained in list", list)
	}
}

func TestDevWorkspaceDoesNotDuplicateTemplatesFromOldProfileCopies(t *testing.T) {
	s, err := OpenDev(t.TempDir(), devSource(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Caster Template", "attack template", "Caster-Template"} {
		path, err := s.filterPath(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := writeJSON(path, Object{"buildProfile": Object{"preferredColors": []any{"R"}}, "buildSpecificOptions": Object{}}); err != nil {
			t.Fatal(err)
		}
	}
	mustCall(t, s, "createFilter", Object{"name": "My Test Filter"})
	list := mustCall(t, s, "listFilters", nil).(Object)["filters"].([]Object)
	if len(list) != 3 {
		t.Fatal("repository templates duplicated by profile copies", list)
	}
	for _, item := range list {
		if item["name"] != "My Test Filter" && item["devTest"] == true {
			t.Fatal("template copy listed as a test filter", item)
		}
	}
	path, _ := s.filterPath("Caster Template")
	if _, err := os.Stat(path); err != nil {
		t.Fatal("listing must not delete old profile data", err)
	}
	if _, err := s.Dispatch("createFilter", Object{"name": "caster-template"}); err == nil {
		t.Fatal("test creation must not collide with template names")
	}
}

func TestDevWorkspaceIsolationAndDirectSourceSave(t *testing.T) {
	root := t.TempDir()
	source := devSource(t)
	personal, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	mustCall(t, personal, "createFilter", Object{"name": "My Filter"})
	mustCall(t, personal, "saveDefaults", Object{"defaults": Object{"preferredColors": []any{"R"}}})
	mustCall(t, personal, "saveStyles", Object{"styles": Object{"currencyA": Object{"text": "#112233"}}})
	personalDefaults, _ := os.ReadFile(filepath.Join(root, "defaults.json"))
	personalStyles, _ := os.ReadFile(filepath.Join(root, "styles.json"))
	personalFile := filepath.Join(root, "filters", "My Filter.json")
	before, _ := os.ReadFile(personalFile)
	dev, err := OpenDev(root, source)
	if err != nil {
		t.Fatal(err)
	}
	list := mustCall(t, dev, "listFilters", nil).(Object)
	if list["devMode"] != true || len(list["filters"].([]Object)) != 2 {
		t.Fatal("expected isolated templates", list)
	}
	if len(c.Obj(mustCall(t, dev, "getDefaults", nil).(Object)["userDefaults"])) != 0 || len(c.Obj(mustCall(t, dev, "getStyles", nil).(Object)["userStyles"])) != 0 {
		t.Fatal("dev workspace inherited personal overrides")
	}
	settings := mustCall(t, dev, "getSettings", nil).(Object)
	if settings["filterPath"] != "" || settings["welcomeComplete"] != true {
		t.Fatal("template workspace should not target the personal game folder", settings)
	}
	for _, action := range []string{"duplicateFilter", "renameFilter", "importJSON", "importFiles", "importFolder", "exportFilter", "exportAllFilters"} {
		if _, err := dev.Dispatch(action, Object{"name": "Caster Template"}); err == nil {
			t.Fatalf("template workspace allowed %s", action)
		}
	}
	cfg := mustCall(t, dev, "getConfig", Object{"name": "Caster Template"}).(Object)
	c.Obj(cfg["buildProfile"])["preferredColors"] = []any{"G"}
	mustCall(t, dev, "saveConfig", Object{"name": "Caster Template", "config": cfg})
	mustCall(t, dev, "saveDefaults", Object{"defaults": Object{"preferredColors": []any{"B"}}})
	mustCall(t, dev, "saveStyles", Object{"styles": Object{"currencyA": Object{"text": "#abcdef"}}})
	reopened, err := OpenDev(root, source)
	if err != nil {
		t.Fatal(err)
	}
	saved := mustCall(t, reopened, "getConfig", Object{"name": "Caster Template"}).(Object)
	if !reflect.DeepEqual(c.Strings(c.Obj(saved["buildProfile"])["preferredColors"]), []string{"G"}) {
		t.Fatal("reopening overwrote authored templates")
	}
	savedTemplates, err := readObject(filepath.Join(source, "templates.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(savedTemplates) != 2 || !reflect.DeepEqual(c.Strings(c.Obj(c.Obj(savedTemplates["Caster Template"])["buildProfile"])["preferredColors"]), []string{"G"}) {
		t.Fatal("Save did not update the repository templates", savedTemplates)
	}
	if !reflect.DeepEqual(savedTemplates["Attack Template"], data.Load[Object]("templates.json")["Attack Template"]) {
		t.Fatal("saving one template changed the other")
	}
	savedDefaults, _ := readObject(filepath.Join(source, "defaults.json"))
	savedStyles, _ := readObject(filepath.Join(source, "styles.json"))
	if !reflect.DeepEqual(c.Strings(savedDefaults["preferredColors"]), []string{"B"}) || savedDefaults["jewellery"] == nil {
		t.Fatal("Save must write defaults with unchanged built-ins preserved", savedDefaults)
	}
	if c.Obj(savedStyles["currencyA"])["text"] != "#abcdef" || savedStyles["gem"] == nil {
		t.Fatal("Save must write styles with unchanged built-ins preserved", savedStyles)
	}
	preview := mustCall(t, dev, "preview", Object{"name": "Caster Template"}).(Object)
	if !strings.Contains(c.Text(preview["output"]), "SetTextColor 171 205 239") {
		t.Fatal("preview did not use the newly saved repository styles")
	}
	if err := personal.writeSource("defaults", Object{}); err == nil {
		t.Fatal("personal mode allowed repository writes")
	}
	after, _ := os.ReadFile(personalFile)
	if string(before) != string(after) {
		t.Fatal("template workspace modified personal filters")
	}
	afterDefaults, _ := os.ReadFile(filepath.Join(root, "defaults.json"))
	afterStyles, _ := os.ReadFile(filepath.Join(root, "styles.json"))
	if string(personalDefaults) != string(afterDefaults) || string(personalStyles) != string(afterStyles) {
		t.Fatal("dev workspace changed personal defaults or styles")
	}
	if len(mustCall(t, personal, "listFilters", nil).(Object)["filters"].([]Object)) != 1 {
		t.Fatal("templates leaked into the personal list")
	}
}

func TestDevSourceResetAndLiveReads(t *testing.T) {
	source := devSource(t)
	baseline := data.Load[Object]("defaults.json")
	baseline["preferredColors"] = []any{"G"}
	if err := writeJSON(filepath.Join(source, "defaults.json"), baseline); err != nil {
		t.Fatal(err)
	}
	dev, err := OpenDev(t.TempDir(), source)
	if err != nil {
		t.Fatal(err)
	}
	mustCall(t, dev, "saveDefaults", Object{"defaults": Object{"preferredColors": []any{"B"}}})
	loaded := mustCall(t, dev, "getDefaults", nil).(Object)
	if len(c.Obj(loaded["userDefaults"])) != 0 || !reflect.DeepEqual(c.Strings(c.Obj(loaded["baseDefaults"])["preferredColors"]), []string{"B"}) {
		t.Fatal("saved source changes must become the new baseline", loaded)
	}
	mustCall(t, dev, "saveDefaults", Object{"defaults": Object{}})
	saved, _ := readObject(filepath.Join(source, "defaults.json"))
	if !reflect.DeepEqual(c.Strings(saved["preferredColors"]), []string{"B"}) {
		t.Fatal("an empty save must not restore the launch-time baseline")
	}
	mustCall(t, dev, "saveDefaults", Object{"defaults": Object{"preferredColors": []any{}}})
	saved, _ = readObject(filepath.Join(source, "defaults.json"))
	if len(c.List(saved["preferredColors"])) != 0 || saved["preferredColors"] == nil {
		t.Fatal("an explicit empty selection was lost")
	}
	mustCall(t, dev, "saveStyles", Object{"styles": Object{"lifeFlask": Object{"text": "#123456"}}})
	loadedStyles := mustCall(t, dev, "getStyles", nil).(Object)
	if len(c.Obj(loadedStyles["userStyles"])) != 0 || c.Obj(c.Obj(loadedStyles["baseStyles"])["lifeFlask"])["text"] != "#123456" {
		t.Fatal("saved styles did not become the new baseline", loadedStyles)
	}
	mustCall(t, dev, "saveStyles", Object{"styles": Object{"lifeFlask": Object{"border": "#abcdef"}}})
	loadedStyles = mustCall(t, dev, "getStyles", nil).(Object)
	style := c.Obj(c.Obj(loadedStyles["baseStyles"])["lifeFlask"])
	if style["text"] != "#123456" || style["border"] != "#abcdef" || len(c.Obj(loadedStyles["userStyles"])) != 0 {
		t.Fatal("a later style save restored the launch-time values", loadedStyles)
	}
	templates, _ := readObject(filepath.Join(source, "templates.json"))
	c.Obj(c.Obj(templates["Caster Template"])["buildProfile"])["preferredColors"] = []any{"G"}
	if err = writeJSON(filepath.Join(source, "templates.json"), templates); err != nil {
		t.Fatal(err)
	}
	cfg := mustCall(t, dev, "getConfig", Object{"name": "Caster Template"}).(Object)
	if !reflect.DeepEqual(c.Strings(c.Obj(cfg["buildProfile"])["preferredColors"]), []string{"G"}) {
		t.Fatal("dev reads should use current repository data, not stale profile copies")
	}
}

func TestDevRequiresExistingSourceFiles(t *testing.T) {
	for _, source := range []string{"", "relative/path", t.TempDir()} {
		if _, err := OpenDev(t.TempDir(), source); err == nil {
			t.Fatalf("accepted invalid repository source directory %q", source)
		}
	}
}
