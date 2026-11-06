package main

import (
	"focolocal-desktop/internal/windows"
	"log"
)

func main() {
	if err := windows.Run(); err != nil {
		log.Fatal(err)
	}
}
