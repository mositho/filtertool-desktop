package main

import (
	"github.com/mositho/filtertool-desktop/internal/basedata"
	"log"
)

func main() {
	if err := basedata.Generate(); err != nil {
		log.Fatal(err)
	}
}
