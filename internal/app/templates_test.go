package app

import (
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
	for _, action := range []string{"createFilter", "duplicateFilter", "renameFilter", "deleteFilter", "importJSON", "importFiles", "importFolder", "exportFilter", "exportAllFilters"} {
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
	if !reflect.DeepEqual(c.Strings(c.Obj(loaded["userDefaults"])["preferredColors"]), []string{"B"}) {
		t.Fatal("saved source changes must remain editable", loaded)
	}
	mustCall(t, dev, "saveDefaults", Object{"defaults": Object{}})
	saved, _ := readObject(filepath.Join(source, "defaults.json"))
	if !reflect.DeepEqual(saved, baseline) {
		t.Fatal("resetting an override should restore the baseline in the repository")
	}
	mustCall(t, dev, "saveDefaults", Object{"defaults": Object{"preferredColors": []any{}}})
	saved, _ = readObject(filepath.Join(source, "defaults.json"))
	if len(c.List(saved["preferredColors"])) != 0 || saved["preferredColors"] == nil {
		t.Fatal("an explicit empty selection was lost")
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
