// Package data embeds the built-in PoE 1 data shipped with Filtertool.
package data

import (
	"embed"
	"encoding/json"
)

//go:embed *.json BaseTypes.csv
var Files embed.FS

func Load[T any](name string) T {
	b, err := Files.ReadFile(name)
	if err != nil {
		panic(err)
	}
	var value T
	if err := json.Unmarshal(b, &value); err != nil {
		panic(err)
	}
	return value
}
