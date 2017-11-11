package argum

import (
	"bytes"
	"log"
	"net/mail"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bradleyjkemp/cupaloy"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

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
	uf, err := prepareStructure(&testusage)
	if err != nil {
		t.Error(err)
	}

	w := bytes.NewBuffer([]byte{})
	uf.writeUsage(w)

	t.Log(w.String())
	outputLines := strings.Split(w.String(), "\n")
	err = cupaloy.New(cupaloy.SnapshotSubdirectory("testdata")).Snapshot(outputLines)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestHelp(t *testing.T) {
	os.Args = []string{"testing", "-s=str", "pos"}
	uf, err := prepareStructure(&testusage)
	if err != nil {
		t.Error(err)
	}

	w := bytes.NewBuffer([]byte{})
	uf.writeHelp(w)
	t.Log(w.String())

	outputLines := strings.Split(w.String(), "\n")
	err = cupaloy.New(cupaloy.SnapshotSubdirectory("testdata")).Snapshot(outputLines)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}
