package lib

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	// "encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tarm/serial"
)

type RotelState struct {
	Balance  string `json:"balance"`
	Bass     string `json:"bass"`
	Display  string `json:"display"`
	Display1 string `json:"display1"`
	Display2 string `json:"display2"`
	Freq     string `json:"freq"`
	Mute     string `json:"mute"`
	Source   string `json:"source"`
	State    string `json:"state"`
	Tone     string `json:"tone"`
	Treble   string `json:"treble"`
	Volume   string `json:"volume"`
}

type RotelMQTTBridge struct {
	SerialPort      *serial.Port
	MQTTClient      mqtt.Client
	RotelDataParser RotelDataParser
	State           *RotelState
}

func CreateSerialPort(serialDevice string) *serial.Port {
	serialConfig := &serial.Config{Name: serialDevice, Baud: 115200}
	serialPort, err := serial.OpenPort(serialConfig)
	if err != nil {
		slog.Error("Could not open port", "serialDevice", serialDevice, "error", err)
		os.Exit(1)
	} else {
		slog.Info("Connected to serial devicen", "serialDevice", serialDevice)
	}
	return serialPort
}

func CreateMQTTClient(mqttBroker string) mqtt.Client {
	slog.Info("Creating MQTT client", "broker", mqttBroker)
	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("Could not connect to broker", "mqttBroker", mqttBroker, "error", token.Error())
		panic(token.Error())
	}
	slog.Info("Connected to MQTT broker", "mqttBroker", mqttBroker)
	return client
}

func NewRotelMQTTBridge(serialPort *serial.Port, mqttClient mqtt.Client) *RotelMQTTBridge {

	bridge := &RotelMQTTBridge{
		SerialPort:      serialPort,
		MQTTClient:      mqttClient,
		RotelDataParser: *NewRotelDataParser(),
		State:           &RotelState{},
	}

	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		"rotel/command/send":       bridge.onCommandSend,
		"rotel/command/initialize": bridge.onInitialize,
	}
	for key, function := range funcs {
		token := mqttClient.Subscribe(key, 0, function)
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
	bridge.SendSerialRequest("get_display1!")
	bridge.SendSerialRequest("get_display2!")
	bridge.SendSerialRequest("get_volume!")
	bridge.SendSerialRequest("get_current_source!")
	bridge.SendSerialRequest("get_current_freq!")
	bridge.SendSerialRequest("get_tone!")
	bridge.SendSerialRequest("get_bass!")
	bridge.SendSerialRequest("get_treble!")
	bridge.SendSerialRequest("get_balance!")
}

var sendMutex sync.Mutex

func (bridge *RotelMQTTBridge) onCommandSend(client mqtt.Client, message mqtt.Message) {
	sendMutex.Lock()
	defer sendMutex.Unlock()

	// Sends command to the Rotel without intermediate parsing
	// Rotel commands are documented here:
	// https://www.rotel.com/sites/default/files/product/rs232/RA12%20Protocol.pdf
	command := string(message.Payload())
	if command != "" {
		bridge.PublishMQTT("rotel/command/send", "", false)
		bridge.SendSerialRequest(command)
	}
}

func (bridge *RotelMQTTBridge) onInitialize(client mqtt.Client, message mqtt.Message) {
	command := string(message.Payload())
	if command != "" {
		bridge.PublishMQTT("rotel/command/initialize", "", false)
		bridge.initialize(true)
	}
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

		slog.Debug("Process Rotel data", "data", data, "command", cmd)

		switch action := cmd[0]; action {
		case "volume":
			bridge.State.Volume = cmd[1]
		case "source":
			bridge.State.Source = cmd[1]
		case "freq":
			bridge.State.Freq = cmd[1]
		case "display":
			bridge.State.Display = cmd[1]
		case "display1":
			bridge.State.Display1 = cmd[1]
		case "display2":
			bridge.State.Display2 = cmd[1]
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
				bridge.initialize(false)
			}
		case "power_off":
			bridge.State.State = "standby"
			bridge.State.Volume = ""
			bridge.State.Source = ""
			bridge.State.Freq = ""
			bridge.State.Display = ""
			bridge.State.Display1 = ""
			bridge.State.Display2 = ""
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
	if strings.HasPrefix(data, "display1=") && len(data) >= len("display1=XX") {
		nextReadCount, _ := strconv.Atoi(data[len("display1="):len("display1=XX")])
		return nextReadCount
	}
	if strings.HasPrefix(data, "display2=") && len(data) >= len("display2=XX") {
		nextReadCount, _ := strconv.Atoi(data[len("display2="):len("display2=XX")])
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
