package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tarm/serial"
)

// Rotel device

type RotelDevice struct {
	SerialPort      *serial.Port
	MQTTClient      mqtt.Client
	RotelDataParser RotelDataParser

	Balance string
	Bass    string
	Display string
	Freq    string
	Mute    string
	Source  string
	State   string
	Tone    string
	Treble  string
	Volume  string
}

func NewRotelDevice(serialDevice string, mqttBroker string) *RotelDevice {

	serialConfig := &serial.Config{Name: serialDevice, Baud: 115200}
	serialPort, err := serial.OpenPort(serialConfig)
	if err != nil {
		log.Fatal(err)
	}

	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	rotelDevice := &RotelDevice{
		SerialPort:      serialPort,
		MQTTClient:      client,
		RotelDataParser: *NewRotelDataParser(),
	}

	funcs := map[string]func(client mqtt.Client, message mqtt.Message){
		"rotel/volume/set": rotelDevice.onVolumeSet,
		"rotel/power/set":  rotelDevice.onPowerSet,
		"rotel/mute/set":   rotelDevice.onMuteSet,
		"rotel/source/set": rotelDevice.onSourceSet,
		"rotel/bass/set":   rotelDevice.onBassSet,
		"rotel/treble/set": rotelDevice.onTrebleSet,
	}
	for key, function := range funcs {
		token := client.Subscribe(key, 0, function)
		token.Wait()
	}
	rotelDevice.initialize()
	return rotelDevice
}

func (rd *RotelDevice) initialize() {
	rd.SendRequest("display_update_auto!")
	rd.SendRequest("get_current_power!")
	rd.SendRequest("get_volume!")
	rd.SendRequest("get_current_source!")
	rd.SendRequest("get_current_freq!")
	rd.SendRequest("get_tone!")
	rd.SendRequest("get_bass!")
	rd.SendRequest("get_treble!")
	rd.SendRequest("get_balance!")
}

func (rd *RotelDevice) onVolumeSet(client mqtt.Client, message mqtt.Message) {
	// up / down / number 1-96 / max / min
	rd.SendRequest(fmt.Sprintf("volume_%s!", string(message.Payload())))
}

func (rd *RotelDevice) onPowerSet(client mqtt.Client, message mqtt.Message) {
	// on / off / toggle
	rd.SendRequest(fmt.Sprintf("power_%s!", string(message.Payload())))
}

func (rd *RotelDevice) onMuteSet(client mqtt.Client, message mqtt.Message) {
	// on / off / "" (=toggle)
	rd.SendRequest(fmt.Sprintf("power_%s!", string(message.Payload())))
}

func (rd *RotelDevice) onSourceSet(client mqtt.Client, message mqtt.Message) {
	// rcd / cd / coax1 / coax2 / opt1 / opt2 / aux1 / aux2 / tuner / phono / usb
	rd.SendRequest(fmt.Sprintf("%s!", string(message.Payload())))
}

func (rd *RotelDevice) onBassSet(client mqtt.Client, message mqtt.Message) {
	// up / down / -10 -- 000 -- +10
	rd.SendRequest(fmt.Sprintf("bass_%s!", string(message.Payload())))
}

func (rd *RotelDevice) onTrebleSet(client mqtt.Client, message mqtt.Message) {
	// up / down / -10 -- 000 -- +10
	rd.SendRequest(fmt.Sprintf("treble_%s!", string(message.Payload())))
}

func (rd *RotelDevice) onBalanceSet(client mqtt.Client, message mqtt.Message) {
	// right / left / L15 -- 000 -- R15
	rd.SendRequest(fmt.Sprintf("treble_%s!", string(message.Payload())))
}

func (rd *RotelDevice) Publish(topic string, message string) {
	token := rd.MQTTClient.Publish(topic, 0, false, message)
	token.Wait()
}

func (rd *RotelDevice) SerialLoop() {
	buf := make([]byte, 128)
	for {
		n, err := rd.SerialPort.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		rd.ProcessData(string(buf[:n]))

		statemap := map[string]string{
			"rotel/balance": rd.Balance,
			"rotel/bass":    rd.Bass,
			"rotel/display": rd.Display,
			"rotel/freq":    rd.Freq,
			"rotel/mute":    rd.Mute,
			"rotel/source":  rd.Source,
			"rotel/state":   rd.State,
			"rotel/tone":    rd.Tone,
			"rotel/treble":  rd.Treble,
			"rotel/volume":  rd.Volume,
		}
		for key, value := range statemap {
			rd.Publish(key, value)
		}
	}
}

var serialWriteMutex sync.Mutex

func (rd *RotelDevice) SendRequest(message string) {
	serialWriteMutex.Lock()
	defer serialWriteMutex.Unlock()

	_, err := rd.SerialPort.Write([]byte(message))
	if err != nil {
		log.Fatal(err)
	}
}

func (rd *RotelDevice) ProcessData(data string) {
	rd.RotelDataParser.HandleParsedData(data)
	for cmd := rd.RotelDataParser.GetNextRotelData(); cmd != nil; cmd = rd.RotelDataParser.GetNextRotelData() {

		switch action := cmd[0]; action {
		case "volume":
			rd.Volume = cmd[1]
		case "source":
			rd.Source = cmd[1]
		case "freq":
			rd.Freq = cmd[1]
		case "display":
			rd.Display = cmd[1]
		case "treble":
			rd.Treble = cmd[1]
		case "bass":
			rd.Bass = cmd[1]
		case "tone":
			rd.Tone = cmd[1]
		case "balance":
			rd.Balance = cmd[1]
		case "mute":
			if cmd[1] == "on/off" {
				rd.SendRequest("get_volume!")
			} else {
				rd.Mute = cmd[1]
			}
		case "power":
			if cmd[1] == "on/standby" {
				rd.SendRequest("get_current_power!")
			} else {
				rd.State = cmd[1]
			}
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
		// nextReadCount := int(data[len("display="):len("display=XXX")][0])
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

func main() {
	rotelDevice := NewRotelDevice("/dev/ttyUSB0", "tcp://localhost:1883")
	rotelDevice.SerialLoop()
}
