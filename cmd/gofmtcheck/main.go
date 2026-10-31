package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	write := flag.Bool("w", false, "format Go source files in place")
	flag.Parse()
	bad := false
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		formatted, err := format.Source(b)
		if err != nil {
			return err
		}
		if !bytes.Equal(b, formatted) {
			if *write {
				if err := os.WriteFile(path, formatted, 0o644); err != nil {
					return err
				}
				fmt.Println(path)
				return nil
			}
			fmt.Println("Run gofmt -w", path)
			bad = true
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if bad {
		os.Exit(1)
	}
}
