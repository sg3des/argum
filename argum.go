package argum

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	//Version is global variable contains version of main application
	Version string
	name    string
	s       *structure
)

//MustParse parse os.Args for struct and fatal if it has error
func MustParse(i interface{}) {
	if err := Parse(i); err != nil {
		fmt.Println(err)
		PrintHelp(1)
	}
}

//Parse os.Args for incomimng struct and return error
func Parse(i interface{}) error {
	if Version != "" && contains(os.Args, "--version") {
		fmt.Println(Version)
		os.Exit(0)
	}

	name = filepath.Base(os.Args[0])
	if filepath.Ext(name) == ".test" {
		return nil
	}

	var err error
	s, err = prepareStructure(i)
	if err != nil {
		return fmt.Errorf("failed prepare structure, %s", err)
	}

	if contains(os.Args[1:], "--help", "-h") {
		s.appendHelpOptions()

		for _, f := range s.fields {
			if f.command && contains(os.Args[1:], f.name) {
				f.s.writeUsageHelp(os.Stdout)
				os.Exit(0)
			}
		}

		s.writeUsageHelp(os.Stdout)
		os.Exit(0)
	}

	_, err = s.parseArgs(os.Args[1:])
	return err
}

//PrintHelp to stdout end exit
func PrintHelp(exitcode int) {
	s.writeUsageHelp(os.Stdout)
	os.Exit(exitcode)
}

func trim(s string) string {
	if len(s) > 1 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			s = strings.Trim(s, "\"")
		}
		if s[0] == '\'' && s[len(s)-1] == '\'' {
			s = strings.Trim(s, "'")
		}
		if s[0] == '`' && s[len(s)-1] == '`' {
			s = strings.Trim(s, "`")
		}
	}

	return s
}

func contains(strslice []string, ss ...string) bool {
	for _, str := range strslice {
		for _, s := range ss {
			if str == s {
				return true
			}
		}
	}
	return false
}

func matchShort(arg string) bool {
	if len(arg) > 1 && arg[0] == '-' && arg[1] != '-' {

		return true
	}

	return false
}

func matchLong(arg string) bool {
	if len(arg) > 2 && arg[0:2] == "--" && arg[2] != '-' {
		return true
	}
	return false
}
