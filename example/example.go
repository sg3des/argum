package main

import (
	"log"

	"github.com/sg3des/argum"
)

var args struct {
	A bool
	B bool
	C bool

	S      string `argum:"req"`
	String string

	Arg        string `argum:"-a"`
	OneMoreArg string `argum:"-o,--onemore"`

	Pos string `argum:"pos"`
}

func main() {
	log.SetFlags(log.Lshortfile)

	argum.Version = "0.1.2"
	argum.MustParse(&args)

	log.Printf("%++v", args)
}
