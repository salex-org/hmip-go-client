package main

import (
	"context"
	"fmt"
	"github.com/salex-org/hmip-go-client/pkg/hmip"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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

	// Notification context for reacting on process termination - used by shutdown function
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Waiting group used to await finishing the shutdown process when stopping
	var wait sync.WaitGroup

	// Loop function for receiving events from HmIP
	wait.Add(1)
	client.RegisterEventHandler(LoggingGenericEventHandler)

	go func() {
		defer wait.Done()
		_ = client.ListenForEvents()
	}()

	// Shutdown function waiting for the SIGTERM notification to start the shutdown process
	wait.Add(1)
	go func() {
		defer wait.Done()
		<-ctx.Done()
		fmt.Printf("\n\U0001F6D1 Shutdown started\n")
		_ = client.StopEventListening()
	}()

	// Wait for all functions to end
	wait.Wait()
	fmt.Printf("\U0001F3C1 Shutdown finished\n")
	os.Exit(0)
}

func LoggingGenericEventHandler(event hmip.Event, origin hmip.Origin) {
	fmt.Printf("\U000023F0 %s - \U0001F4AC: Event of type %s from origin %s (type %s)", time.Now(), event.GetType(), origin.GetID(), origin.GetType())
	switch specialEvent := event.(type) {
	case hmip.DeviceChangedEvent:
		device := specialEvent.GetDevice()
		channels := device.GetFunctionalChannels()
		fmt.Printf(" for device %s (type %s)\n", device.GetName(), device.GetType())
		for _, channel := range channels {
			switch specialChannel := channel.(type) {
			case hmip.SwitchChannel:
				fmt.Printf("\U000023F0 %s - \U0001F4AC %s: Aktiv %t\n", device.GetLastUpdated(), device.GetName(), specialChannel.IsSwitchedOn())
			case hmip.SwitchMeasuringChannel:
				fmt.Printf("\U000023F0 %s - \U0001F4AC %s: Aktiv %t, Consumption %.2fWh\n", device.GetLastUpdated(), device.GetName(), specialChannel.IsSwitchedOn(), specialChannel.GetCurrentPowerConsumption())
			case hmip.ClimateSensorChannel:
				fmt.Printf("\U000023F0 %s - \U0001F4AC %s: \U0001F321 %.1f°C, \U0001F4A7 %d%%, \U0001F328 %.2f\n",
					device.GetLastUpdated(),
					device.GetName(),
					specialChannel.GetActualTemperature(),
					specialChannel.GetHumidity(),
					specialChannel.GetVapourAmount())
			}
		}
	case hmip.GroupChangedEvent:
		fmt.Printf(" for group %s (type %s)\n", specialEvent.GetGroup().GetName(), specialEvent.GetGroup().GetType())
	default:
		fmt.Printf("\n")
	}
}
