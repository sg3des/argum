package main

import (
	"log"

	"github.com/sg3des/argum"
)

var args struct {
	A bool `help:"a option, enable something"`
	B bool `help:"if true, then something will happen"`
	C bool `help:"c enable something"`

	S      string `argum:"req,str0|str1|str2" help:"required value for something"`
	String string `help:"set string value"`

	Arg        string `argum:"-a" help:"optional you may set Arg variable"`
	OneMoreArg string `argum:"-o,--onemore" default:"some-value" help:"one more arg"`

	Pos string `argum:"pos,debug|normal|fast" default:"normal" help:"mode"`
}

func main() {
	log.SetFlags(log.Lshortfile)

	argum.Version = "0.1.2"
	argum.MustParse(&args)

	log.Printf("%++v", args)
}
