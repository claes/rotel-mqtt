package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tarm/serial"
)

// MQTT bridge

// type MQTTBridge struct {
// 	MQTTClient mqtt.Client
// }

// func NewMQTTBridge() *MQTTBridge {
// 	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
// 	client := mqtt.NewClient(opts)
// 	if token := client.Connect(); token.Wait() && token.Error() != nil {
// 		panic(token.Error())
// 	}

// 	token := client.Subscribe("rotel/volume", 0, onVolumeUpdate)
// 	token.Wait()

// 	return &MQTTBridge{
// 		MQTTClient: client,
// 	}
// }

// func (mb *MQTTBridge) Publish(topic string, message string) {
// 	token := mb.MQTTClient.Publish(topic, 0, false, message)
// 	token.Wait()
// }

// Rotel device

type RotelDevice struct {
	SerialPort      *serial.Port
	MQTTClient      mqtt.Client
	RotelDataParser RotelDataParser

	Volume  string
	Source  string
	Freq    string
	Mute    string
	State   string
	Display string
}

func (rd *RotelDevice) onVolumeUpdate(client mqtt.Client, message mqtt.Message) {
	fmt.Printf("%s\t%s\n", message.Topic(), message.Payload())
}

func (rd *RotelDevice) onVolumeSet(client mqtt.Client, message mqtt.Message) {
	rd.SendRequest("volume_up!")
}

func NewRotelDevice() *RotelDevice {

	serialConfig := &serial.Config{Name: "/dev/ttyUSB0", Baud: 115200}
	serialPort, err := serial.OpenPort(serialConfig)
	if err != nil {
		log.Fatal(err)
	}

	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	rotelDevice := &RotelDevice{
		SerialPort:      serialPort,
		MQTTClient:      client,
		RotelDataParser: *NewRotelDataParser(),
	}

	token := client.Subscribe("rotel/volume", 0, rotelDevice.onVolumeUpdate)
	token.Wait()

	token = client.Subscribe("rotel/volume/set", 0, rotelDevice.onVolumeSet)
	token.Wait()

	return rotelDevice
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

		rd.Publish("rotel/volume", rd.Volume)
		// fmt.Printf("Volume: %s\n", rd.Volume)
		// fmt.Printf("Source: %s\n", rd.Source)
		// fmt.Printf("Freq: %s\n", rd.Freq)
		// fmt.Printf("Display: %s\n", rd.Display)
		// fmt.Printf("Len Display: %d\n", len(rd.Display))
		// fmt.Printf("State: %s\n", rd.State)
		// fmt.Printf("Mute: %s\n", rd.Mute)
		// fmt.Printf("\n")
	}
}

//var serialMutex sync.Mutex

func (rd *RotelDevice) SendRequest(message string) {
	// serialMutex.Lock()
	// defer serialMutex.Unlock()

	// send
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
		case "mute":
			if cmd[1] == "on/off" {
				rd.SendRequest("get_volume!")
			} else {
				rd.Mute = cmd[1]
			}
		case "power":
			if cmd[1] == "on/standby" {
				rd.SendRequest("get_power!")
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
	rotelDevice := NewRotelDevice()
	rotelDevice.SerialLoop()
}
