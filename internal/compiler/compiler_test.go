package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebCompilerParity(t *testing.T) {
	// Compare both compilers against the same captured base data. Weekly CSV
	// updates must not invalidate the oracle by changing only the Go inputs.
	weaponBytes, err := os.ReadFile("testdata/data/weapons.json")
	if err != nil {
		t.Fatal(err)
	}
	var referenceWeapons []Weapon
	if err = json.Unmarshal(weaponBytes, &referenceWeapons); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("testdata/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 10 {
		t.Fatal("missing reference fixtures")
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			b, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var cfg Object
			if err = json.Unmarshal(b, &cfg); err != nil {
				t.Fatal(err)
			}
			compiler := New(Obj(cfg["_defaults"]), Obj(cfg["_styles"]), "")
			// The captured web compiler predates starter-support defaults. Keep
			// that historical baseline here; current defaults have separate tests.
			compiler.Defaults["gemCallouts"] = Object{}
			compiler.weapons = referenceWeapons
			got, err := compiler.Compile(cfg)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(strings.TrimSuffix(file, ".json") + ".filter")
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				out := filepath.Join(t.TempDir(), "actual.filter")
				_ = os.WriteFile(out, []byte(got), 0600)
				a, b := strings.Split(got, "\n"), strings.Split(string(want), "\n")
				for i := 0; i < len(a) && i < len(b); i++ {
					if a[i] != b[i] {
						t.Fatalf("first difference line %d: got %q; want %q (actual %s)", i+1, a[i], b[i], out)
					}
				}
				t.Fatalf("output length mismatch: got %d, want %d", len(got), len(want))
			}
		})
	}
}

func TestOverridesAndValidation(t *testing.T) {
	c := New(nil, Object{"currencyA": Object{"text": "#123456"}}, "custom-sounds")
	out, err := c.Compile(Object{"buildProfile": Object{}, "buildSpecificOptions": Object{}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SetTextColor 18 52 86") || !strings.Contains(out, "custom-sounds/") {
		t.Fatal("overrides not applied")
	}
	if _, err := c.Compile(Object{"buildProfile": Object{"weaponPhases": []any{Object{"minAreaLevel": float64(20), "maxAreaLevel": float64(10)}}}, "buildSpecificOptions": Object{}}); err == nil {
		t.Fatal("accepted reversed range")
	}
}
