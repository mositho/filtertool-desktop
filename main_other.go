//go:build !windows && !(linux && linuxdesktop)

package main

import "fmt"

func main() {
	fmt.Println("Filtertool's desktop UI targets Windows. Run backend checks with go test ./... or build with npm run build.")
}
