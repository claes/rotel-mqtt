package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	// "encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tarm/serial"
)

var debug *bool

type RotelState struct {
	Balance string `json:"balance"`
	Bass    string `json:"bass"`
	Display string `json:"display"`
	Freq    string `json:"freq"`
	Mute    string `json:"mute"`
	Source  string `json:"source"`
	State   string `json:"state"`
	Tone    string `json:"tone"`
	Treble  string `json:"treble"`
	Volume  string `json:"volume"`
}

type RotelMQTTBridge struct {
	SerialPort      *serial.Port
	MQTTClient      mqtt.Client
	RotelDataParser RotelDataParser
	State           *RotelState
}

func NewRotelMQTTBridge(serialDevice string, mqttBroker string) *RotelMQTTBridge {

	serialConfig := &serial.Config{Name: serialDevice, Baud: 115200}
	serialPort, err := serial.OpenPort(serialConfig)
	if err != nil {
		log.Fatal(err)
	} else if *debug {
		fmt.Printf("Connected to serial device: %s\n", serialDevice)
	}

	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	} else if *debug {
		fmt.Printf("Connected to MQTT broker: %s\n", mqttBroker)
	}

	bridge := &RotelMQTTBridge{
		SerialPort:      serialPort,
		MQTTClient:      client,
		RotelDataParser: *NewRotelDataParser(),
		State:           &RotelState{},
	}

	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		"rotel/volume/set": bridge.onVolumeSet,
		"rotel/power/set":  bridge.onPowerSet,
		"rotel/mute/set":   bridge.onMuteSet,
		"rotel/source/set": bridge.onSourceSet,
		"rotel/bass/set":   bridge.onBassSet,
		"rotel/treble/set": bridge.onTrebleSet,
	}
	for key, function := range funcs {
		token := client.Subscribe(key, 0, function)
		token.Wait()
	}
	time.Sleep(2 * time.Second)
	bridge.initialize(true)
	return bridge
}

func (bridge *RotelMQTTBridge) initialize(askPower bool) {
	if askPower {
		// to avoid recursion when initializing after power on
		bridge.SendSerialRequest("get_current_power!")
	}
	bridge.SendSerialRequest("display_update_auto!")
	bridge.SendSerialRequest("get_display!")
	bridge.SendSerialRequest("get_volume!")
	bridge.SendSerialRequest("get_current_source!")
	bridge.SendSerialRequest("get_current_freq!")
	bridge.SendSerialRequest("get_tone!")
	bridge.SendSerialRequest("get_bass!")
	bridge.SendSerialRequest("get_treble!")
	bridge.SendSerialRequest("get_balance!")
}

func (bridge *RotelMQTTBridge) onVolumeSet(client mqtt.Client, message mqtt.Message) {
	// up / down / number 1-96 / max / min
	p := string(message.Payload())
	//TODO handle number
	if p == "up" || p == "down" || p == "max" || p == "min" {
		bridge.PublishMQTT("rotel/volume/set", "", false)
		bridge.SendSerialRequest(fmt.Sprintf("volume_%s!", p))
	}
}

func (bridge *RotelMQTTBridge) onPowerSet(client mqtt.Client, message mqtt.Message) {
	// on / off / toggle
	p := string(message.Payload())
	if p == "on" || p == "off" {
		bridge.PublishMQTT("rotel/power/set", "", false)
		bridge.SendSerialRequest(fmt.Sprintf("power_%s!", p))
	}
}

func (bridge *RotelMQTTBridge) onMuteSet(client mqtt.Client, message mqtt.Message) {
	// on / off / "" (=toggle)
	p := string(message.Payload())
	if p == "on" || p == "off" {
		bridge.PublishMQTT("rotel/mute/set", "", false)
		bridge.SendSerialRequest(fmt.Sprintf("mute_%s!", p))
	}
}

func (bridge *RotelMQTTBridge) onSourceSet(client mqtt.Client, message mqtt.Message) {
	// rcd / cd / coax1 / coax2 / opt1 / opt2 / aux1 / aux2 / tuner / phono / usb
	p := string(message.Payload())
	if p == "rcd" || p == "cd" || p == "coax1" || p == "coax2" ||
		p == "opt1" || p == "opt2" || p == "aux1" || p == "aux2" ||
		p == "tuner" || p == "phono" || p == "usb" {
		bridge.PublishMQTT("rotel/source/set", "", false)
		bridge.SendSerialRequest(fmt.Sprintf("%s!", string(message.Payload())))
	}
}

func (bridge *RotelMQTTBridge) onBassSet(client mqtt.Client, message mqtt.Message) {
	// up / down / -10 -- 000 -- +10
	bridge.SendSerialRequest(fmt.Sprintf("bass_%s!", string(message.Payload())))
}

func (bridge *RotelMQTTBridge) onTrebleSet(client mqtt.Client, message mqtt.Message) {
	// up / down / -10 -- 000 -- +10
	bridge.SendSerialRequest(fmt.Sprintf("treble_%s!", string(message.Payload())))
}

func (bridge *RotelMQTTBridge) onBalanceSet(client mqtt.Client, message mqtt.Message) {
	// right / left / L15 -- 000 -- R15
	bridge.SendSerialRequest(fmt.Sprintf("treble_%s!", string(message.Payload())))
}

func (bridge *RotelMQTTBridge) PublishMQTT(topic string, message string, retained bool) {
	token := bridge.MQTTClient.Publish(topic, 0, retained, message)
	token.Wait()
}

func (bridge *RotelMQTTBridge) SerialLoop() {
	buf := make([]byte, 128)
	for {
		n, err := bridge.SerialPort.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		bridge.ProcessRotelData(string(buf[:n]))

		jsonState, err := json.Marshal(bridge.State)
		if err != nil {
			continue
		}
		bridge.PublishMQTT("rotel/state", string(jsonState), true)
	}
}

var serialWriteMutex sync.Mutex

func (bridge *RotelMQTTBridge) SendSerialRequest(message string) {
	serialWriteMutex.Lock()
	defer serialWriteMutex.Unlock()

	_, err := bridge.SerialPort.Write([]byte(message))
	if err != nil {
		log.Fatal(err)
	}
}

func (bridge *RotelMQTTBridge) ProcessRotelData(data string) {
	bridge.RotelDataParser.HandleParsedData(data)
	for cmd := bridge.RotelDataParser.GetNextRotelData(); cmd != nil; cmd = bridge.RotelDataParser.GetNextRotelData() {

		if *debug {
			fmt.Printf("cmd: %v\n", cmd)
		}

		switch action := cmd[0]; action {
		case "volume":
			bridge.State.Volume = cmd[1]
		case "source":
			bridge.State.Source = cmd[1]
		case "freq":
			bridge.State.Freq = cmd[1]
		case "display":
			bridge.State.Display = cmd[1]
		case "treble":
			bridge.State.Treble = cmd[1]
		case "bass":
			bridge.State.Bass = cmd[1]
		case "tone":
			bridge.State.Tone = cmd[1]
		case "balance":
			bridge.State.Balance = cmd[1]
		case "mute":
			if cmd[1] == "on/off" {
				bridge.SendSerialRequest("get_volume!")
			} else {
				bridge.State.Mute = cmd[1]
			}
		case "power":
			if cmd[1] == "on" {
				bridge.State.State = cmd[1]
				bridge.initialize(false)
			} else if cmd[1] == "standby" {
				bridge.State.State = cmd[1]
			}
		case "power_off":
			bridge.State.State = "standby"
			bridge.State.Volume = ""
			bridge.State.Source = ""
			bridge.State.Freq = ""
			bridge.State.Display = ""
			bridge.State.Treble = ""
			bridge.State.Bass = ""
			bridge.State.Tone = ""
			bridge.State.Balance = ""
		}
	}
}

// Rotel data parser

type RotelDataParser struct {
	RotelDataQueue   [][]string
	NextKeyValuePair string
}

func NewRotelDataParser() *RotelDataParser {
	return &RotelDataParser{
		RotelDataQueue:   [][]string{},
		NextKeyValuePair: "",
	}
}

func (rdp *RotelDataParser) GetNextRotelData() []string {
	if len(rdp.RotelDataQueue) > 0 {
		retVal := rdp.RotelDataQueue[0]
		rdp.RotelDataQueue = rdp.RotelDataQueue[1:]
		return retVal
	} else {
		return nil
	}
}

func (rdp *RotelDataParser) PushKeyValuePair(keyValuePair string) {
	keyValue := strings.Split(keyValuePair, "=")
	rdp.RotelDataQueue = append(rdp.RotelDataQueue, keyValue)
}

func (rdp *RotelDataParser) PushRotelData(rotelData []string) {
	rdp.RotelDataQueue = append(rdp.RotelDataQueue, rotelData)
}

func (rdp *RotelDataParser) ComputeFixedLengthDataToRead(data string) int {
	if strings.HasPrefix(data, "display=") && len(data) >= len("display=XXX") {
		nextReadCount, _ := strconv.Atoi(data[len("display="):len("display=XXX")])
		return nextReadCount
	}
	return 0
}

func (rdp *RotelDataParser) HandleParsedData(data string) {
	for _, c := range data {
		fixedLengthDataToRead := rdp.ComputeFixedLengthDataToRead(rdp.NextKeyValuePair)
		if fixedLengthDataToRead > 0 {
			s := rdp.NextKeyValuePair + string(c)
			startIndex := len("display=XXX") + 1
			if strings.HasPrefix(s, "display=") && (len(s)-startIndex) >= fixedLengthDataToRead {
				value := s[startIndex : startIndex+fixedLengthDataToRead]
				rdp.PushRotelData([]string{"display", value})
				rdp.NextKeyValuePair = ""
			} else {
				rdp.NextKeyValuePair += string(c)
			}
		} else if "!" == string(c) {
			rdp.PushKeyValuePair(rdp.NextKeyValuePair)
			rdp.NextKeyValuePair = ""
		} else {
			rdp.NextKeyValuePair += string(c)
		}
	}
}

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

	bridge := NewRotelMQTTBridge(*serialDevice, *mqttBroker)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Started\n")
	go bridge.SerialLoop()
	<-c
	fmt.Printf("Shut down\n")
	os.Exit(0)
}
