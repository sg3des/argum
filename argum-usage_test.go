package argum

import (
	"bytes"
	"net/mail"
	"os"
	"testing"
	"time"
)

var testusage struct {
	S        string `argum:"req,str|str1|str2"`
	String   string
	Name     string `argum:"-n,--name" help:"name of something with multiline long long long long text, and text and some text"`
	Duration time.Duration
	Mail     []*mail.Address `argum:"-m"`

	Pos   string   `argum:"pos,req" help:"positional argument more more more text and text text text"`
	Slice []string `argum:"pos" help:"slice arguments"`
}

func TestUsage(t *testing.T) {
	os.Args = []string{"testing", "-s=str1", "pos"}
	err := Parse(&testusage)
	if err != nil {
		t.Error(err)
	}

	w := bytes.NewBuffer([]byte{})
	Usage(w)
	t.Log(w.String())
	if w.Len() == 0 {
		t.Error("failed get usage line")
	}
}

func TestHelp(t *testing.T) {
	os.Args = []string{"testing", "-s=str", "pos"}
	err := Parse(&testusage)
	if err != nil {
		t.Error(err)
	}

	w := bytes.NewBuffer([]byte{})
	Help(w)
	t.Log(w.String())
	if len(w.String()) == 0 {
		t.Error("failed get help")
	}
}
