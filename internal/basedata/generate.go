// Package basedata generates desktop reference data from FilterBlade's PoE 1 CSV.
package basedata

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strconv"
	"strings"
)

func rows(b []byte) ([]map[string]string, error) {
	r := csv.NewReader(bytes.NewReader(b))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, errors.New("base data has no rows")
	}
	header := records[0]
	found := map[string]bool{}
	for _, s := range header {
		found[strings.TrimPrefix(s, "\ufeff")] = true
	}
	for _, key := range []string{"Class", "BaseType", "DropLevel", "Game:APS"} {
		if !found[key] {
			return nil, errors.New("base data is missing column " + key)
		}
	}
	out := []map[string]string{}
	for _, row := range records[1:] {
		entry := map[string]string{}
		for i, v := range row {
			if i < len(header) {
				entry[strings.TrimPrefix(header[i], "\ufeff")] = v
			}
		}
		out = append(out, entry)
	}
	return out, nil
}
func ValidateCSV(b []byte) error {
	parsed, err := rows(b)
	if err != nil {
		return err
	}
	if len(parsed) < 100 {
		return errors.New("unexpectedly small base data")
	}
	return nil
}
func write(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}
func Generate() error {
	b, err := os.ReadFile("internal/data/BaseTypes.csv")
	if err != nil {
		return err
	}
	records, err := rows(b)
	if err != nil {
		return err
	}
	referenceBytes, err := os.ReadFile("internal/data/reference.json")
	if err != nil {
		return err
	}
	var ref map[string]any
	if err = json.Unmarshal(referenceBytes, &ref); err != nil {
		return err
	}
	classes := map[string][]string{}
	weapons := []map[string]any{}
	weaponClasses := []string{"One Hand Axes", "Two Hand Axes", "Bows", "Claws", "Daggers", "Rune Daggers", "One Hand Maces", "Sceptres", "Two Hand Maces", "Staves", "Warstaves", "One Hand Swords", "Thrusting One Hand Swords", "Two Hand Swords", "Wands", "Fishing Rods"}
	isWeapon := func(name string) bool {
		if name == "Shields" {
			return true
		}
		for _, v := range weaponClasses {
			if v == name {
				return true
			}
		}
		return false
	}
	for _, row := range records {
		class, base := row["Class"], row["BaseType"]
		if class == "" || base == "" {
			continue
		}
		classes[class] = append(classes[class], base)
		if isWeapon(class) {
			drop, _ := strconv.ParseFloat(row["DropLevel"], 64)
			w := map[string]any{"baseType": base, "itemClass": class, "dropLevel": drop}
			for key, column := range map[string]string{"aps": "Game:APS", "damageFrom": "Game:Damage From", "damageTo": "Game:Damage To", "dps": "Game:DPS", "crit": "Game:Crit"} {
				if n, err := strconv.ParseFloat(row[column], 64); err == nil {
					w[key] = n
				}
			}
			implicits := []string{}
			for _, column := range []string{"Game:Implicit 1", "Game:Implicit 2"} {
				if row[column] != "" {
					implicits = append(implicits, row[column])
				}
			}
			w["implicits"] = implicits
			weapons = append(weapons, w)
		}
	}
	names := []string{}
	for name := range classes {
		names = append(names, name)
		sort.Strings(classes[name])
	}
	sort.Strings(names)
	// Keep source CSV order: it determines the stable order of weapon cutoff groups.
	ref["itemClasses"] = names
	ref["baseTypesByClass"] = classes
	ref["skillGems"] = classes["Skill Gems"]
	ref["supportGems"] = classes["Support Gems"]
	ref["weaponClasses"] = weaponClasses
	for _, v := range ref["sounds"].([]any) {
		v.(map[string]any)["hasFile"] = false
	}
	if err = write("internal/data/reference.json", ref); err != nil {
		return err
	}
	return write("internal/data/weapons.json", weapons)
}
