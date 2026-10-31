package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func TestTemplateWorkspaceIsolationAndExplicitExport(t *testing.T) {
	root := t.TempDir()
	personal, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	mustCall(t, personal, "createFilter", Object{"name": "My Filter"})
	personalFile := filepath.Join(root, "filters", "My Filter.json")
	before, _ := os.ReadFile(personalFile)
	templates, err := OpenTemplates(root)
	if err != nil {
		t.Fatal(err)
	}
	list := mustCall(t, templates, "listFilters", nil).(Object)
	if list["templateMode"] != true || len(list["filters"].([]Object)) != 2 {
		t.Fatal("expected isolated templates", list)
	}
	settings := mustCall(t, templates, "getSettings", nil).(Object)
	if settings["filterPath"] != "" || settings["welcomeComplete"] != true {
		t.Fatal("template workspace should not target the personal game folder", settings)
	}
	for _, action := range []string{"createFilter", "duplicateFilter", "renameFilter", "deleteFilter", "importJSON", "importFiles", "importFolder", "saveDefaults", "saveStyles"} {
		if _, err := templates.Dispatch(action, Object{"name": "Caster Template"}); err == nil {
			t.Fatalf("template workspace allowed %s", action)
		}
	}
	cfg := mustCall(t, templates, "getConfig", Object{"name": "Caster Template"}).(Object)
	c.Obj(cfg["buildProfile"])["preferredColors"] = []any{"G"}
	mustCall(t, templates, "saveConfig", Object{"name": "Caster Template", "config": cfg})
	reopened, err := OpenTemplates(root)
	if err != nil {
		t.Fatal(err)
	}
	saved := mustCall(t, reopened, "getConfig", Object{"name": "Caster Template"}).(Object)
	if !reflect.DeepEqual(c.Strings(c.Obj(saved["buildProfile"])["preferredColors"]), []string{"G"}) {
		t.Fatal("reopening overwrote authored templates")
	}
	var exported Object
	templates.SaveShareFile = func(name, text string) (string, error) {
		if name != "templates.json" {
			t.Fatal("wrong bundle filename", name)
		}
		return "templates.json", json.Unmarshal([]byte(text), &exported)
	}
	mustCall(t, templates, "saveTemplateBundle", nil)
	if len(exported) != 2 || !reflect.DeepEqual(c.Strings(c.Obj(c.Obj(exported["Caster Template"])["buildProfile"])["preferredColors"]), []string{"G"}) {
		t.Fatal("export did not include saved templates", exported)
	}
	if _, err := personal.Dispatch("saveTemplateBundle", nil); err == nil {
		t.Fatal("personal mode allowed template source export")
	}
	after, _ := os.ReadFile(personalFile)
	if string(before) != string(after) {
		t.Fatal("template workspace modified personal filters")
	}
	if len(mustCall(t, personal, "listFilters", nil).(Object)["filters"].([]Object)) != 1 {
		t.Fatal("templates leaked into the personal list")
	}
}
