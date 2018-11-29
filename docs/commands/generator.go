package main

import (
	"log"
	"strings"

	"github.com/spf13/cobra/doc"
	banjo "github.com/thetatoken/ukulele/cmd/banjo/cmd"
	ukulele "github.com/thetatoken/ukulele/cmd/ukulele/cmd"
)

func generateBanjoDoc(filePrepender, linkHandler func(string) string) {
	var all = banjo.RootCmd
	err := doc.GenMarkdownTreeCustom(all, "./wallet/", filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}
}

func generateUkuleleDoc(filePrepender, linkHandler func(string) string) {
	var all = ukulele.RootCmd
	err := doc.GenMarkdownTreeCustom(all, "./ledger/", filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	filePrepender := func(filename string) string {
		return ""
	}

	linkHandler := func(name string) string {
		return strings.ToLower(name)
	}

	generateBanjoDoc(filePrepender, linkHandler)
	generateUkuleleDoc(filePrepender, linkHandler)
	Walk()
}
