package main

import (
	"log"
	"strings"

	"github.com/spf13/cobra/doc"
	theta "github.com/thetatoken/theta/cmd/theta/cmd"
	thetacli "github.com/thetatoken/theta/cmd/thetacli/cmd"
)

func generateThetaCLIDoc(filePrepender, linkHandler func(string) string) {
	var all = thetacli.RootCmd
	err := doc.GenMarkdownTreeCustom(all, "./wallet/", filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}
}

func generateThetaDoc(filePrepender, linkHandler func(string) string) {
	var all = theta.RootCmd
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

	generateThetaCLIDoc(filePrepender, linkHandler)
	generateThetaDoc(filePrepender, linkHandler)
	Walk()
}
