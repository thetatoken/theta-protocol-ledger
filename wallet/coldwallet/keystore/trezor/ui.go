package trezor

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/crypto/ssh/terminal"
)

const (
	PinMatrixRequestType_Current   = 1
	PinMatrixRequestType_NewFirst  = 2
	PinMatrixRequestType_NewSecond = 3
)

const PIN_MATRIX_DESCRIPTION = `Use the numeric keypad to describe number positions. The layout is:
    7 8 9
    4 5 6
    1 2 3`

type TrezorUI struct {
	pinmatrixShown bool
	promptShown    bool
	alwaysPrompt   bool
}

func NewTrezorUI(always bool) *TrezorUI {
	return &TrezorUI{alwaysPrompt: always}
}

func (ui *TrezorUI) ButtonRequest() {
	if !ui.promptShown {
		fmt.Println("Please confirm action on your Trezor device")
	}

	if !ui.alwaysPrompt {
		ui.promptShown = true
	}
}

func (ui *TrezorUI) GetPin(code PinMatrixRequestType) string {
	var desc string
	if code == PinMatrixRequestType_Current {
		desc = "current PIN"
	} else if code == PinMatrixRequestType_NewFirst {
		desc = "new PIN"
	} else if code == PinMatrixRequestType_NewSecond {
		desc = "new PIN again"
	} else {
		desc = "PIN"
	}
	if !ui.pinmatrixShown {
		fmt.Println(PIN_MATRIX_DESCRIPTION)
		if !ui.alwaysPrompt {
			ui.pinmatrixShown = true
		}
	}
	for {
		pin := prompt(fmt.Sprintf("Please enter %v: ", desc))
		// except click.Abort:
		//     raise Cancelled from None
		fmt.Println()
		if _, err := strconv.Atoi(pin); err != nil {
			fmt.Println("Non-numerical PIN provided, please try again")
		} else {
			return pin
		}
	}
}

func (ui *TrezorUI) GetPassphrase() string {
	if os.Getenv("PASSPHRASE") != "" {
		fmt.Println("Passphrase required. Using PASSPHRASE environment variable.")
		return os.Getenv("PASSPHRASE")
	}

	for {
		passphrase := prompt("Passphrase required")
		second := prompt("Confirm your passphrase")
		if passphrase == second {
			return passphrase
		}
		fmt.Println("\nPassphrase did not match. Please try again.")
	}
	//     except click.Abort:
	//         raise Cancelled from None
}

func prompt(p string) string {
	fmt.Print(p)
	pin, _ := terminal.ReadPassword(0)
	return string(pin)
}
