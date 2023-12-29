package main

import (
	"encoding/json"
	"fmt"
	"github.com/salex-org/hmip-go-client/pkg/hmip"
	"log"
)

const (
	ColorRedBold = "\033[1;31m"
	ColorOff     = "\033[0m"
)

func main() {
	client, err := hmip.GetClient()
	if err != nil {
		log.Fatalf("\U0001F6AB %sFailed%s to create client: %v\n", ColorRedBold, ColorOff, err)
	}
	var state hmip.State
	state, err = client.LoadCurrentState()
	if err != nil {
		log.Fatalf("\U0001F6AB %sFailed%s to load state: %v\n", ColorRedBold, ColorOff, err)
	}
	var output []byte
	output, err = json.Marshal(state)
	if err != nil {
		log.Fatalf("\U0001F6AB %sFailed%s to marshal state: %v\n", ColorRedBold, ColorOff, err)
	}
	fmt.Print(string(output))
}
