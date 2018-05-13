[![Build Status](https://travis-ci.org/sg3des/argum.svg?branch=master)](https://travis-ci.org/sg3des/argum)

# Argum

Argum is package for parse arguments into struct, inspired by [alexflint/go-arg](https://github.com/alexflint/go-arg).

WARNING! work in progress, backward compatible is not guaranteed!

```shell
go get github.com/sg3des/argum
```

# Description

Setting up available arguments using tags:

 * `argum:"-s"` - set short signature
 * `argum:"--str"` - set long signature
 * `argum:"-s,--str"` - set short and long signature
 * `argum:"req"` or `argum:"required"`  - required argument
 * `argum:"pos"` or `argum:"positional"` - positional argument
 * `argum:"oneof"` - this keyword work only on internal struct, user can select only one of nested fields, itself structure ignored from command line
 * `argum:"emb"` or `argum:"embedded"` - its keyword work only on for internal struct, and indicates that the struct name should be ignored
 * `help:"some help"` - help description for this option
 * `default:"value"` - default value
 * if struct field not have tag *argum*, then parse it automate

Argum, use 3 key tags for parse structure - *argum*, *help*, *default* - it's more convenient.

# Usage

```go
var args struct {
	A string                      //parsed to -a
	Arg string                    //parsed to --arg
	SomeArg string `argum:"-s"`   //only -s
	OneMoreArg string `argum:"-o,--onemore"`  //both keys: -o, --onemore
}
argum.MustParse(&args)
```

### Set software version

```go
argum.Version = "some version"
argum.MustParse(&args)
```

### Default values

```go
var args struct {
	String string    `default:"some string"`
	Slice  []string  `default:"one,two,three"`
	IntSlice []int `argum:"--int" default:"0,2,3"`
}
```

Default value for slice automatic split by comma character

### Joined boolean arguments

```go
var args struct {
	A bool
	B bool
	C bool `argum:"-c"`
	D bool
	E bool 
}
argum.MustParse(&args)
```

This options can be specified as `./example -abcde`, and each of listed will be set to `true`

### Internal structs and `oneof` keyword

```go
var args struct {
	Command Commands `argum:"req,oneof" help:"select main command"`

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

```

Keyword `oneof` is means user should chose only one of nested fields. These example structures will provide next command lines: 

	./example ping 127.0.0.1
	./example -d ping 127.0.0.1
	./example ping 127.0.0.1 -c4
	./example ping 127.0.0.1 -c4 listen 8000

	./example echo 8080
	./example echo 8080 -d
	./example echo 8080 listen 8000


### Embedding structs

```go
var args struct {
	Ping `argum:"emb"`
}

type Ping struct {
	IP string `argum:"pos,req"`
	Count int    `argum:"-c" help:"count of packets"`
}
```

Keyword `emb` or `embedded` indicates that the structure name should be ignored. These example will provide next command lines:

	./example 127.0.0.1
	./example 127.0.0.1 -c 4


### Help and Usage output

```go
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
argum.Version = "example version 0.1.2"
argum.MustParse(&args)
```

```
usage: example [-abc] -s=[str0|str1|str2] [--string=<s>] [-a=<s>] [-o=<s>] [debug|normal|fast]

positional:
  pos                     mode [default: normal] [debug|normal|fast]

options:
  -a                      a option, enable something
  -b                      if true, then something will happen
  -c                      c enable something
  -s=[str0|str1|str2]     required value for something [str0|str1|str2]
      --string=<s>        set string value
  -a=<s>                  optional you may set Arg variable
  -o, --onemore=<s>       one more arg [default: some-value]
  -h, --help              display this help and exit
      --version           display version and exit

```
