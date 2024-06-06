package lib

import (
	"strconv"
	"strings"
	// "encoding/json"
)

// Rotel data parser

type RotelDataParser struct {
	RotelDataQueue   [][]string
	NextKeyValuePair string
	buffer           string
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
	keyValue := strings.Split(strings.TrimSuffix(keyValuePair, "!"), "=")
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

func matchCommand(input string) bool {
	return strings.Count(input, "=") == 1 &&
		strings.HasSuffix(input, "!")
}

func matchDisplay(pattern, input string) *string {

	pos := strings.Index(pattern, "=")
	if pos == -1 {
		return nil
	}

	prefix := pattern[:pos+1]
	lengthFormat := pattern[pos+1:]

	startIndex := strings.Index(input, prefix)
	if startIndex == -1 {
		return nil
	}

	numberStartIndex := startIndex + len(prefix)

	if numberStartIndex+len(lengthFormat) > len(input) {
		return nil
	}

	numberStr := input[numberStartIndex : numberStartIndex+len(lengthFormat)]

	count, err := strconv.Atoi(numberStr)
	if err != nil {
		return nil
	}

	substringStartIndex := numberStartIndex + len(lengthFormat) + 1 //1 for comma

	substringEndIndex := substringStartIndex + count
	if substringEndIndex > len(input) {
		return nil
	}

	result := input[substringStartIndex:substringEndIndex]
	return &result
}

func (rdp *RotelDataParser) match(s string) (bool, string) {

	if matchCommand(s) {
		return true, s
	} else {
		displayMatch := matchDisplay("display1=XX", s)
		if displayMatch != nil {
			return true, "display1=" + *displayMatch + "!"
		}
		displayMatch = matchDisplay("display2=XX", s)
		if displayMatch != nil {
			return true, "display2=" + *displayMatch + "!"
		}
		displayMatch = matchDisplay("display=XXX", s)
		if displayMatch != nil {
			return true, "display=" + *displayMatch + "!"
		}
	}
	return false, s
}

func (rdp *RotelDataParser) HandleParsedData(data string) {
	rdp.buffer += string(data)
	next := ""
outer:
	for len(rdp.buffer) > 0 {
		for i, c := range rdp.buffer {
			next += string(c)
			match, matchedString := rdp.match(next)
			if match {
				rdp.PushKeyValuePair(matchedString)
				next = ""
				rdp.buffer = rdp.buffer[i+1:]
				continue outer
			}
		}
		break
	}
}

func (rdp *RotelDataParser) HandleParsedDataOld(data string) {
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
