package main

import (
	"bufio"
	"fmt"
	"github.com/salex-org/hmip-go-client/pkg/hmip"
	"os"
	"strings"
)

const (
	ColorCyanBold  = "\033[1;36m"
	ColorRedBold   = "\033[1;31m"
	ColorGreenBold = "\033[1;32m"
	ColorOff       = "\033[0m"
)

func main() {
	config, err := hmip.GetConfig()
	if err != nil {
		fmt.Printf("\U0001F6AB %sFailed%s to create new client config: %v\n", ColorRedBold, ColorOff, err)
		return
	}

	fmt.Printf("Registering new client in the Homematic IP Cloud\n")
	fmt.Printf("\U0000251C Client Name: ")
	config.ClientName = commandLineInput()
	fmt.Printf("\U0000251C Access Point SGTIN: ")
	config.AccessPointSGTIN = commandLineInput()
	fmt.Printf("\U00002570 PIN: ")
	config.PIN = commandLineInput()

	err = config.RegisterClient(func() {
		fmt.Printf("\U0001F6CE Please press the %sblue button%s on the access point to confirm the client registration\n", ColorCyanBold, ColorOff)
	})
	if err != nil {
		fmt.Printf("\U0001F6AB %sFailed%s to register new client %s%s%s: %v\n", ColorRedBold, ColorOff, ColorCyanBold, config.ClientName, ColorOff, err)
		return
	}

	fmt.Printf("\U0001F3C1 %sSuccessfully%s registered new client %s%s%s\n", ColorGreenBold, ColorOff, ColorCyanBold, config.ClientName, ColorOff)
	fmt.Printf("\U0001F3AB Device ID: %s\n", config.DeviceID)
	fmt.Printf("\U0001F3AB Client ID: %s\n", config.ClientID)
	fmt.Printf("\U0001F511 Client Auth Token: %s\n", config.ClientAuthToken)
	fmt.Printf("\U0001F511 Auth Token: %s\n", config.AuthToken)
}

func commandLineInput() string {
	input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSuffix(strings.TrimSuffix(input, "\n"), "\r")
}
