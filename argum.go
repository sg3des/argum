package argum

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// Description is global variable contains description of application, output on help or arguments error
	Description string
	// Version is global variable contains version of main application
	Version string
	name    string
	s       *structure
)

// MustParse parse os.Args for struct and fatal if it has error
func MustParse(i interface{}) {
	if err := Parse(i); err != nil {
		fmt.Println(err)
		PrintHelp(1)
	}
}

// Parse os.Args for incomimng struct and return error
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
		// INFO: temporary hidden help for specify command, as now output all help information
		// for _, f := range s.fields {
		// 	if f.command && contains(os.Args[1:], f.name) {
		// 		f.s.writeUsageHelp(os.Stdout)
		// 		os.Exit(0)
		// 	}
		// }

		s.writeUsageHelp(os.Stdout)
		os.Exit(0)
	}

	_, err = s.parseArgs(os.Args[1:])
	return err
}

// PrintHelp to stdout end exit
func PrintHelp(exitcode int) {
	s.writeUsageHelp(os.Stdout)
	os.Exit(exitcode)
}

func splitArg(s string) (string, []string) {
	if matchEscape(s) {
		s = trim(s)
	}

	switch {
	// case matchEscape(s):
	// 	s = trim(s)
	// 	if strings.ContainsAny(s, ",/ ") {
	// 		return "", strings.Split(s, ",")
	// 	}
	case strings.Contains(s, "="):
		ss := strings.SplitN(s, "=", 2)
		return ss[0], splitValues(ss[1])
	case matchShort(s) && len(s) > 2:
		return s[:2], splitValues(s[2:])
	case strings.Contains(s, ","):
		return "", strings.Split(s, ",")
	}

	return s, nil
}

func splitValues(s string) (vals []string) {
	if matchEscape(s) {
		vals = []string{s}
	} else {
		vals = strings.Split(s, ",")
	}
	return vals
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

func matchShort(s string) bool {
	return len(s) > 1 && s[0] == '-' && s[1] != '-'
}

func matchSortBooleans(s string) bool {
	return len(s) > 2 && s[0] == '-' && s[1] != '-' && s[2] > 0x40 && s[2] < 0x7B
}

func matchLong(s string) bool {
	return len(s) > 2 && s[0:2] == "--" && s[2] != '-'
}

func matchEscape(s string) bool {
	if len(s) > 0 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			return true
		}
		if s[0] == '\'' && s[len(s)-1] == '\'' {
			return true
		}
		if s[0] == '`' && s[len(s)-1] == '`' {
			return true
		}
	}
	return false
}
