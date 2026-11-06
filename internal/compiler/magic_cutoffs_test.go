package compiler

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func magicMaxima(t *testing.T, c *Compiler, options Object, want map[string]float64) {
	t.Helper()
	out := c.equipment("Magic", c.section(options, "magicItems"))
	for key, maximum := range want {
		found := false
		for _, block := range strings.Split(out, "\n\n") {
			if strings.Contains(block, "Width == "+key[:1]+"\n") && strings.Contains(block, "Height == "+key[2:]+"\n") {
				found = true
				if !strings.Contains(block, "AreaLevel <= "+fmt.Sprint(maximum)+"\n") {
					t.Fatalf("%s: expected maximum %v: %s", key, maximum, block)
				}
			}
		}
		if !found {
			t.Fatal("missing magic inventory size", key)
		}
	}
}

func TestMagicItemsUseOneDefaultWithAdvancedOverrides(t *testing.T) {
	c := New(Object{"magicItems": Object{"maxAreaLevel": float64(31), "rules": Object{
		"1x1": Object{"maxAreaLevel": float64(7)},
		"2x4": Object{"maxAreaLevel": float64(42)},
		"1x3": Object{"maxAreaLevel": float64(0)},
	}}}, nil, "")
	magicMaxima(t, c, Object{}, map[string]float64{"1x1": 7, "2x2": 31, "2x4": 42, "1x3": 0})
	magicMaxima(t, c, Object{"magicItems": Object{"maxAreaLevel": float64(18)}}, map[string]float64{"1x1": 7, "2x2": 18, "2x4": 42})
}

func TestLegacySharedMagicCutoffsBecomeAdvancedRulesWithoutMutatingInputs(t *testing.T) {
	old := Object{"magicItems": Object{
		"smallMaxAreaLevel": float64(17), "bigMaxAreaLevel": float64(6),
		"rules": Object{"1x3": Object{"maxAreaLevel": float64(11), "size": float64(33)}},
	}}
	before := Clone(old)
	c := New(old, nil, "")
	magicMaxima(t, c, Object{}, map[string]float64{"1x1": 17, "2x2": 17, "1x3": 11, "2x4": 6})
	clean := CleanRuleDefaults(old)
	magic := Obj(clean["magicItems"])
	if magic["maxAreaLevel"] != float64(17) || magic["smallMaxAreaLevel"] != nil || magic["bigMaxAreaLevel"] != nil {
		t.Fatal("legacy split was not normalized", magic)
	}
	if Obj(Obj(magic["rules"])["1x3"])["size"] != float64(33) || !reflect.DeepEqual(old, before) {
		t.Fatal("normalization lost rule tuning or mutated the input", old, clean)
	}
}

func TestImportedMagicSplitRemainsEffectiveUntilUnifiedCutoffIsSet(t *testing.T) {
	c := New(nil, nil, "")
	options := Object{"magicItems": Object{"smallMaxAreaLevel": float64(8), "bigMaxAreaLevel": float64(3)}}
	magicMaxima(t, c, options, map[string]float64{"1x1": 8, "2x2": 8, "1x3": 3, "2x4": 3})
	Obj(options["magicItems"])["maxAreaLevel"] = float64(19)
	rules := Obj(Obj(c.Defaults["magicItems"])["rules"])
	magicMaxima(t, c, options, map[string]float64{
		"1x1": 19, "2x2": 19,
		"1x3": Num(Obj(rules["1x3"])["maxAreaLevel"], 19),
		"2x4": Num(Obj(rules["2x4"])["maxAreaLevel"], 19),
	})
}
