package main

import (
	"testing"
)

func TestTerminated(t *testing.T) {
	r := NewRotelDataParser()
	r.HandleParsedData("source=coax2!freq=44.1!")

	rotelData := r.GetNextRotelData()
	if rotelData[0] != "source" || rotelData[1] != "coax2" {
		t.Error("Expected 'source, coax2', got ", rotelData)
	}

	rotelData = r.GetNextRotelData()
	if rotelData[0] != "freq" || rotelData[1] != "44.1" {
		t.Error("Expected 'freq, 44.1', got ", rotelData)
	}
}

func TestFixedLength(t *testing.T) {
	r := NewRotelDataParser()
	r.HandleParsedData("display=010,0123456789A")

	rotelData := r.GetNextRotelData()
	if rotelData[0] != "display" || rotelData[1] != "0123456789" {
		t.Error("Expected 'display, 0123456789', got ", rotelData)
	}
}

// func TestFixedLengthSplit(t *testing.T) {
// 	r := NewRotelDataParser()
// 	r.HandleParsedData("display=010,01234")
// 	r.HandleParsedData("56789A")

// 	rotelData := r.GetNextRotelData()
// 	if rotelData[0] != "display" || rotelData[1] != "0123456789" {
// 		t.Error("Expected 'display, 0123456789', got ", rotelData)
// 	}
// }

// func TestMixed(t *testing.T) {
// 	r := NewRotelDataParser()
// 	r.HandleParsedData("disp")
// 	r.HandleParsedData("lay=010,01234")
// 	r.HandleParsedData("56789")
// 	r.HandleParsedData("source=coax2!fr")
// 	r.HandleParsedData("eq=44.1!")

// 	rotelData := r.GetNextRotelData()
// 	if rotelData[0] != "display" || rotelData[1] != "0123456789" {
// 		t.Error("Expected 'display, 0123456789', got ", rotelData)
// 	}

// 	rotelData = r.GetNextRotelData()
// 	if rotelData[0] != "source" || rotelData[1] != "coax2" {
// 		t.Error("Expected 'source, coax2', got ", rotelData)
// 	}

// 	rotelData = r.GetNextRotelData()
// 	if rotelData[0] != "freq" || rotelData[1] != "44.1" {
// 		t.Error("Expected 'freq, 44.1', got ", rotelData)
// 	}
// }

