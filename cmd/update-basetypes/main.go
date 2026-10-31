package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mositho/filtertool-desktop/internal/basedata"
)

func main() {
	client := http.Client{Timeout: 60 * time.Second}
	res, err := client.Get("https://raw.githubusercontent.com/NeverSinkDev/FilterBlade-Public-Assets/main/FbPoe1Configs/BaseTypes.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal(res.Status)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, 10*1024*1024+1))
	if err != nil {
		log.Fatal(err)
	}
	if len(b) > 10*1024*1024 {
		log.Fatal("base data is too large")
	}
	if err = basedata.ValidateCSV(b); err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile("internal/data/BaseTypes.csv", b, 0644); err != nil {
		log.Fatal(err)
	}
	if err = basedata.Generate(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Updated PoE 1 base data and regenerated references.")
}
