package main

import (
	"flag"
	"fmt"
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

	bridge := lib.NewRotelMQTTBridge(lib.CreateSerialPort(*serialDevice),
		lib.CreateMQTTClient(*mqttBroker))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Started\n")
	go bridge.SerialLoop()
	<-c
	fmt.Printf("Shut down\n")
	os.Exit(0)
}
