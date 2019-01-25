package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bgentry/speakeasy"
	isatty "github.com/mattn/go-isatty"
)

var buf *bufio.Reader

func GetPassword(prompt string) (password string, err error) {
	if inputIsTty() {
		password, err = speakeasy.Ask(prompt)
	} else {
		password, err = stdinLine()
	}
	return
}

func GetConfirmation() (confirmation string, err error) {
	confirmation, err = stdinLine()
	return
}

func inputIsTty() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

func stdinLine() (string, error) {
	if buf == nil {
		buf = bufio.NewReader(os.Stdin)
	}
	line, err := buf.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func Error(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
	os.Exit(1)
}
