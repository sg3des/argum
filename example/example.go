package main

import (
	"fmt"

	argum "github.com/sg3des/argum"
)

var args struct {
	Ping *Ping `help:"some ping"`
	Echo *Echo `help:"open local port"`

	Value string `argum:"pos" help:"Some string value"`
	Debug bool   `argum:"-d,--debug" help:"Enable debug mode"`
}

type Ping struct {
	IP    string `argum:"req,pos" help:"ip address"`
	Count int    `argum:"-c" help:"count of packets"`
}

type Echo struct {
	Port int `argum:"req,pos" help:"port number"`
}

func main() {
	argum.Version = "0.1.2"
	argum.MustParse(&args)

	fmt.Printf("%+v\n", args)
	switch {
	case args.Ping != nil:
		fmt.Printf("%+v\n", args.Ping)
	case args.Echo != nil:
		fmt.Printf("%+v\n", args.Echo)
	}
}
