package main

import (
	"fmt"
	"log"

	"github.com/sg3des/argum"
)

var args struct {
	Mode *Mode `help:"mode"` //mode is always positional
	// Normal *Mode `help:"Normal mode"`
	// Fast   *Mode `help:"Fast mode"`

	//StringValue string `argum:"pos" help:"Some string value"`
	Debug bool `argum:"-d,--debug" help:"Enable debug mode"`
	Port  int  `argum:"-p" help:"Port number"`
}

var modes = []string{"normal", "fast"}

//Mode is internal struct object used to separate logic
type Mode struct {
	mode string //argum ignore non-exported fields
	Addr string `argum:"pos" help:"ip-address"`
}

//Check is function used to verify incoming arguments and should initialize object if this necessary. `Check` method has a higher priority than `Variants`
func (m *Mode) Check(s string) error {
	if m != nil {
		return fmt.Errorf("Mode has been selected")
	}

	for _, ss := range modes {
		if ss == s {
			if m == nil {
				m = new(Mode)
			}
			m.mode = s
			return nil
		}
	}

	return fmt.Errorf("Unsuitable mode %s, available modes: %s", s, modes)
}

//Variants is semi-automatic alternative of manual method `Check`, should return slice of strings contains suitable values to initialize fill fields of this struct. Variants has a lower priority than `Check`, and will not be used if `Check` method declared.
func (m *Mode) Variants() []string {
	return modes
}

//Usage is custom functuon returned usage string. Do not use \n and \r symbols here.
func (m *Mode) Usage() string {
	return "Custom usage string"
}

//Help is custom function, returned help string.
func (m *Mode) Help() string {
	return "Custom help message about how to use this"
}

func main() {

	log.SetFlags(log.Lshortfile)

	argum.Version = "0.1.2"
	argum.MustParse(&args)

	// prettyStruct.Print("", args)
}
