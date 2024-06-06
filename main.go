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

func maintest() {

	r := lib.NewRotelDataParser()

	//r.NextKeyValuePair = "balance=000!"
	r.NextKeyValuePair = "display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!"
	//                                12345678901234567890
	//r.NextKeyValuePair = "display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!"
	//r.NextKeyValuePair = "display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax"

	//	fmt.Printf("Next rotel data %d\n", len(r.GetNextRotelData()))

	fmt.Printf("nkv %d %s\n", len(r.RotelDataQueue), r.NextKeyValuePair)
	r.HandleParsedData("balance=000!")
	fmt.Printf("nkv %d %s\n", len(r.RotelDataQueue), r.NextKeyValuePair)
	fmt.Printf("Next rotel data %d\n", len(r.GetNextRotelData()))
	fmt.Printf("Next rotel data %d\n", len(r.GetNextRotelData()))
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
