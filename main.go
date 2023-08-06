package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/tarm/serial"
)

// // MQTT bridge

// type MQTTBridge struct {
// 	MQTTClient mqtt.Client
// }

// func NewMQTTBridge() *MQTTBridge {
// 	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
// 	client := mqtt.NewClient(opts)
// 	if token := client.Connect(); token.Wait() && token.Error() != nil {
// 		panic(token.Error())
// 	}

// 	return &MQTTBridge{
// 		MQTTClient : client,
// 	}
// }

// func (mb *MQTTBridge) Publish(message string) {

// 	for {
// 		//currentTime := time.Now().Format(time.RFC3339)
// 		token := client.Publish("time/topic", 0, false, message)
// 		token.Wait()
// 		fmt.Println("Published:", currentTime)
// 		time.Sleep(1 * time.Second)
// 	}
// }

// Rotel device

type RotelDevice struct {
	rotelDataParser RotelDataParser
	Volume          string
	Source          string
	Freq            string
	Mute            string
	State           string
	Display         string
}

func NewRotelDevice() *RotelDevice {
	return &RotelDevice{
		rotelDataParser: *NewRotelDataParser(),
	}
}

func (rd *RotelDevice) SerialLoop() {
	c := &serial.Config{Name: "/dev/ttyUSB0", Baud: 115200}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 128)
	for {
		n, err := s.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Print(string(buf[:n]))
		rd.ProcessData(string(buf[:n]))
		fmt.Printf("Volume: %s\n", rd.Volume)
		fmt.Printf("Source: %s\n", rd.Source)
		fmt.Printf("Freq: %s\n", rd.Freq)
		fmt.Printf("Display: %s\n", rd.Display)
		fmt.Printf("Len Display: %d\n", len(rd.Display))
		fmt.Printf("State: %s\n", rd.State)
		fmt.Printf("Mute: %s\n", rd.Mute)
		fmt.Printf("\n")
	}
}

//var serialMutex sync.Mutex

func (rd *RotelDevice) SendRequest(message string) {
	// serialMutex.Lock()
	// defer serialMutex.Unlock()

	// send
}

func (rd *RotelDevice) ProcessData(data string) {
	rd.rotelDataParser.HandleParsedData(data)
	for cmd := rd.rotelDataParser.GetNextRotelData(); cmd != nil; cmd = rd.rotelDataParser.GetNextRotelData() {

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

	// c := &serial.Config{Name: "/dev/ttyUSB0", Baud: 115200}
	// s, err := serial.OpenPort(c)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// buf := make([]byte, 128)
	// for {
	// 	n, err := s.Read(buf)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Print(string(buf[:n]))
	// }
}
