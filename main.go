package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	// "encoding/json"

	"github.com/claes/rotel-mqtt/lib"
)

var debug *bool

func printHelp() {
	fmt.Println("Usage: rotel-mqtt [OPTIONS]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}

func main() {
	serialDevice := flag.String("serial", "/dev/ttyUSB0", "Serial device path")
	mqttBroker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	help := flag.Bool("help", false, "Print help")
	debug = flag.Bool("debug", false, "Debug logging")
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	serialPort, err := lib.CreateSerialPort(*serialDevice)
	if err != nil {
		slog.Error("Error creating serial device", "error", err, "serialDevice", *serialDevice)
	}
	mqttClient, err := lib.CreateMQTTClient(*mqttBroker)
	if err != nil {
		slog.Error("Error creating mqtt client", "error", err, "broker", *mqttBroker)
	}
	bridge := lib.NewRotelMQTTBridge(serialPort, mqttClient)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Started\n")
	go bridge.SerialLoop()
	<-c
	fmt.Printf("Shut down\n")
	os.Exit(0)
}
