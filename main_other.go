//go:build !windows

package main

import "fmt"

func main() {
	fmt.Println("Filtertool's desktop UI targets Windows. Run backend checks with go test ./... or build with npm run build.")
}
