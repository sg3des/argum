package main

import (
	"fmt"
	"log"

	argum "github.com/sg3des/argum"
)

var args struct {
	Command Commands `argum:"req,selection" help:"select main command"`

	Listen *Echo `help:"optional internal struct"`

	Value string `argum:"pos" help:"Some string value"`
	Debug bool   `argum:"-d,--debug" help:"Enable debug mode"`
}

type Commands struct {
	Ping *Ping  `help:"some ping"`
	Echo *Echo  `help:"open local port"`
	Str  string `argum:"pos" help:"simple string value instead struct"`
}

type Ping struct {
	IP    string `argum:"req,pos" help:"ip address"`
	Count int    `argum:"-c" help:"count of packets"`
}

type Echo struct {
	Port int `argum:"req,pos" help:"port number"`
}

func main() {
	log.SetFlags(log.Lshortfile)

	argum.Version = "0.1.2"
	argum.MustParse(&args)

	fmt.Printf("%+v\n", args)

	switch {
	case args.Command.Ping != nil:
		fmt.Printf("PING: %+v\n", args.Command.Ping)
	case args.Command.Echo != nil:
		fmt.Printf("ECHO: %+v\n", args.Command.Echo)
	case args.Command.Str != "":
		fmt.Printf("STRING: %s\n", args.Command.Str)
	}

	if args.Listen != nil {
		fmt.Printf("LISTEN: %+v\n", args.Listen)
	}

}
