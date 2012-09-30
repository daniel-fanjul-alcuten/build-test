package build

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestSourceFileName(t *testing.T) {
	t.Parallel()

	r := NewResourceManager(3, 5)
	s := NewSourceFile(r, "foo")

	if s.Name() != "foo" {
		t.Error("incorrect s.Name()", s.Name())
	}
}

func TestSourceFileExists(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile("", "go-build-test-")
	if err != nil {
		t.Fatal(err)
	}
	name := file.Name()
	defer os.Remove(name)
	defer file.Close()
	n, err := file.Write([]byte{'f', 'o', 'o'})
	if n != 3 || err != nil {
		t.Fatal(err)
	}

	r := NewResourceManager(3, 5)
	s := NewSourceFile(r, name)
	s.Connect()

	stat := s.Stat()
	if (stat.time == time.Time{}) {
		t.Error("incorrect stat.time")
	}
	if stat.size != 3 {
		t.Error("incorrect stat.size", stat.size)
	}
	if stat.err != nil {
		t.Error("incorrect stat.err", stat.err)
	}
	if s.stat == nil {
		t.Error("incorrect s.stat", s.stat)
	}
	if s.stat.time != stat.time {
		t.Error("incorrect s.stat.time", s.stat.time)
	}
	if s.stat.size != stat.size {
		t.Error("incorrect s.stat.size", s.stat.size)
	}
	if s.stat.err != stat.err {
		t.Error("incorrect s.stat.err", s.stat.err)
	}

	read := s.Read()
	if string(read.buf.Data()) != "foo" {
		t.Error("incorrect read.buf.Data()", read.buf.Data())
	}
	if read.err != nil {
		t.Error("incorrect read.err", read.err)
	}
	if s.read == nil {
		t.Error("incorrect s.read", s.read)
	}
	if s.read.buf != read.buf {
		t.Error("incorrect s.read.buf", s.read.buf)
	}
	if s.read.err != read.err {
		t.Error("incorrect s.read.err", s.read.err)
	}

	s.Disconnect()
	s.Connected()
}

func TestSourceFileDoesNotExist(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile("", "go-build-test-")
	if err != nil {
		t.Fatal(err)
	}
	name := file.Name()
	err = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove(name)
	if err != nil {
		t.Fatal(err)
	}

	r := NewResourceManager(3, 5)
	s := NewSourceFile(r, name)
	s.Connect()

	stat := s.Stat()
	if (stat.time != time.Time{}) {
		t.Error("incorrect stat.time", stat.time)
	}
	if stat.size != 0 {
		t.Error("incorrect stat.size", stat.size)
	}
	if stat.err == nil {
		t.Error("incorrect stat.err", stat.err)
	}
	if s.stat == nil {
		t.Error("incorrect s.stat", s.stat)
	}
	if s.stat.time != stat.time {
		t.Error("incorrect s.stat.time", s.stat.time)
	}
	if s.stat.size != stat.size {
		t.Error("incorrect s.stat.size", s.stat.size)
	}
	if s.stat.err != stat.err {
		t.Error("incorrect s.stat.err", s.stat.err)
	}

	read := s.Read()
	if read.buf != nil {
		t.Error("incorrect read.buf", read.buf)
	}
	if read.err == nil {
		t.Error("incorrect read.err", read.err)
	}
	if s.read == nil {
		t.Error("incorrect s.read", s.read)
	}
	if s.read.buf != read.buf {
		t.Error("incorrect s.read.buf", s.read.buf)
	}
	if s.read.err != read.err {
		t.Error("incorrect s.read.err", s.read.err)
	}

	s.Disconnect()
	s.Connected()
}
