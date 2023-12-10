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
	client.RegisterEventHandler(LoggingCLimateSensordHandler, hmip.EVENT_TYPE_DEVICE_CHANGED)
	client.RegisterEventHandler(LoggingPluggableSwitchActiveHandler, hmip.EVENT_TYPE_DEVICE_CHANGED)
	client.RegisterEventHandler(LoggingPluggableSwitchMeasuringHandler, hmip.EVENT_TYPE_DEVICE_CHANGED)
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
	fmt.Printf("\U000023F0 %s - \U0001F4AC: Event of type %s", time.Now(), event.Type)
	if event.Device != nil {
		fmt.Printf(" for device %s (type %s)", event.Device.Name, event.Device.Type)
	}
	if event.Group != nil {
		fmt.Printf(" for group %s (type %s)", event.Group.Name, event.Group.Type)
	}
	fmt.Printf(" from origin %s (type %s)\n", origin.ID, origin.Type)
}

func LoggingCLimateSensordHandler(event hmip.Event, _ hmip.Origin) {
	for _, channel := range event.GetFunctionalChannels(hmip.DEVICE_TYPE_TEMPERATURE_HUMIDITY_SENSOR_OUTDOOR, hmip.CHANNEL_TYPE_CLIMATE_SENSOR) {
		fmt.Printf("\U000023F0 %s - \U0001F4AC %s: \U0001F321 %.1f°C, \U0001F4A7 %d%%, \U0001F328 %.2f\n",
			event.Device.LastStatusUpdate.Time,
			event.Device.Name,
			channel.Temperature,
			channel.Humidity,
			channel.VapourAmount)
	}
}

func LoggingPluggableSwitchActiveHandler(event hmip.Event, _ hmip.Origin) {
	for _, channel := range event.GetFunctionalChannels(hmip.DEVICE_TYPE_PLUGABLE_SWITCH, hmip.CHANNEL_TYPE_SWITCH) {
		fmt.Printf("\U000023F0 %s - \U0001F4AC %s: Aktiv %t\n", event.Device.LastStatusUpdate.Time, event.Device.Name, channel.SwitchedOn)
	}
}

func LoggingPluggableSwitchMeasuringHandler(event hmip.Event, _ hmip.Origin) {
	for _, channel := range event.GetFunctionalChannels(hmip.DEVICE_TYPE_PLUGABLE_SWITCH_MEASURING, hmip.CHANNEL_TYPE_SWITCH_MEASURING) {
		fmt.Printf("\U000023F0 %s - \U0001F4AC %s: Aktiv %t, Consumption %.2fWh\n", event.Device.LastStatusUpdate.Time, event.Device.Name, channel.SwitchedOn, channel.CurrentPowerConsumption)
	}
}
