package lib

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

func TestFixedLengthSplit(t *testing.T) {
	r := NewRotelDataParser()
	r.HandleParsedData("display=010,01234")
	r.HandleParsedData("56789A")

	rotelData := r.GetNextRotelData()
	if rotelData[0] != "display" || rotelData[1] != "0123456789" {
		t.Error("Expected 'display, 0123456789', got ", rotelData)
	}
}

func TestMixed(t *testing.T) {
	r := NewRotelDataParser()
	r.HandleParsedData("disp")
	r.HandleParsedData("lay=010,01234")
	r.HandleParsedData("56789")
	r.HandleParsedData("source=coax2!fr")
	r.HandleParsedData("eq=44.1!")

	rotelData := r.GetNextRotelData()
	if rotelData[0] != "display" || rotelData[1] != "0123456789" {
		t.Error("Expected 'display, 0123456789', got ", rotelData)
	}

	rotelData = r.GetNextRotelData()
	if rotelData[0] != "source" || rotelData[1] != "coax2" {
		t.Error("Expected 'source, coax2', got ", rotelData)
	}

	rotelData = r.GetNextRotelData()
	if rotelData[0] != "freq" || rotelData[1] != "44.1" {
		t.Error("Expected 'freq, 44.1', got ", rotelData)
	}
}

func TestFoo(t *testing.T) {
	r := NewRotelDataParser()
	//r.HandleParsedData(" PCM    volume=39!source=opt1!fr")

	r.HandleParsedData("power=on!")
	found := false
	for cmd := r.GetNextRotelData(); cmd != nil; cmd = r.GetNextRotelData() {
		switch action := cmd[0]; action {
		case "power":
			found = true
		}
	}
	if !found {
		t.Error("Did not find power")
	}
}

func TestDisplay(t *testing.T) {
	r := matchDisplay("display1=XX", "display1=04,abcde")
	if *r != "abcd" {
		t.Error("Failed match, expected abcd, was " + *r)
	}

	r = matchDisplay("display1=XXX", "display1=004,abcde")
	if *r != "abcd" {
		t.Error("Failed match, expected abcd, was " + *r)
	}

}

func TestTerminatedNew(t *testing.T) {
	r := NewRotelDataParser()
	r.HandleParsedData("source=coax2!freq=44.1!")
	r.HandleParsedData("display=010,01234")
	r.HandleParsedData("56789A")
	r.HandleParsedData("display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!power=on!display_update=auto!display=040,  COAX1      VOL 39  BASS 0     TREB 0  display1=20,  COAX1      VOL 39 display2=20, BASS 0     TREB 0  volume=39!source=coax1!freq=off!tone=on!bass=000!treble=000!balance=000!")

	rotelData := r.GetNextRotelData()
	if rotelData[0] != "source" || rotelData[1] != "coax2" {
		t.Error("Expected 'source, coax2', got ", rotelData)
	}

	rotelData = r.GetNextRotelData()
	if rotelData[0] != "freq" || rotelData[1] != "44.1" {
		t.Error("Expected 'freq, 44.1', got ", rotelData)
	}

	rotelData = r.GetNextRotelData()
	if rotelData[0] != "display" || rotelData[1] != "0123456789" {
		t.Error("Expected 'display, 0123456789', got ", rotelData)
	}
	rotelData = r.GetNextRotelData()
	if rotelData[0] != "display1" || rotelData[1] != "  COAX1      VOL 39 " {
		t.Error("Expected 'display1,   COAX1      VOL 39 ', got ", rotelData)
	}

}
