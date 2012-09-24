package main

import (
	. "dfanjul/build"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var (
	res *ResourceManager
)

func main() {

	m := flag.Int64("m", 128*1024*1024, "memory")
	d := flag.Uint("d", 1, "disk ops")
	flag.Parse()

	res = NewResourceManager(*m, *d)

	err := work()
	if err != nil {
		log.Fatal(err)
	}
}

func work() error {

	start := time.Now()

	entries, err := ioutil.ReadDir("foo")
	if err != nil {
		return err
	}

	err = os.Mkdir("bar1", 0777)
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	err = os.Mkdir("bar2", 0777)
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	inits, works, dones := make(chan bool), 0, make(chan error, 1024)
	go func() {
		nodes := make([]Node, 0, 3*len(entries))
		for _, entry := range entries {
			if entry.Mode()&os.ModeType == 0 {
				file := NewSourceFile(res, "foo/"+entry.Name())
				work1 := NewCopyFile(res, file, "bar1/"+entry.Name())
				go func(worker Worker) {
					dones <- worker.Work()
				}(work1)
				work2 := NewCopyFile(res, file, "bar2/"+entry.Name())
				go func(worker Worker) {
					dones <- worker.Work()
				}(work2)
				nodes = append(nodes, file, work1, work2)
				works += 2
			}
			//time.Sleep(time.Millisecond)
		}
		for _, node := range nodes {
			node.Connected()
		}
		inits <- true
	}()

	init, done, ticker := false, 0, time.Tick(701*time.Millisecond)
	for !init {
		select {
		case init = <-inits:
		case err := <-dones:
			done++
			if err != nil {
				clear()
				return err
			}
		case <-ticker:
			info(start, init, done, works)
		}
	}
	for done < works {
		select {
		case err := <-dones:
			done++
			if err != nil {
				clear()
				return err
			}
		case <-ticker:
			info(start, init, done, works)
		}
	}

	info(start, init, done, works)
	fmt.Println()
	return nil
}

var (
	lastLenLine = 0
	init_format = "\r%v/%v+ nodes (~%0.2f%%), %v mem (%0.2f%%), %v/%v/%v disk (%0.2f%%); %v + ~%v ~= %v"
	flow_format = "\r%v/%v nodes (%0.2f%%), %v mem (%0.2f%%), %v/%v/%v disk (%0.2f%%); %v + ~%v ~= %v"
)

func info(start time.Time, init bool, done, total int) {
	var format string
	if !init {
		format = init_format
	} else {
		format = flow_format
	}
	var prog float32
	if total > 0 {
		prog = float32(done) * 100.0 / float32(total)
	} else {
		prog = 0
	}
	muse, mmax := res.Mem()
	mprog := float32(muse) * 100.0 / float32(mmax)
	dstats, dreads, dwrites, dmax := res.Disk()
	dprog := float32(dstats+dreads+dwrites) * 100.0 / float32(dmax)
	tdone := time.Since(start)
	ttotal := tdone * time.Duration(total) / time.Duration(done)
	tdone, ttotal = tdone/time.Second*time.Second, ttotal/time.Second*time.Second
	tpending := ttotal - tdone
	line := fmt.Sprintf(format, done, total, prog, muse, mprog, dstats, dreads, dwrites, dprog, tdone, tpending, ttotal)
	lenLine := len(line)
	if lastLenLine > lenLine {
		line += strings.Repeat(" ", lastLenLine-lenLine)
	}
	fmt.Print(line)
	lastLenLine = lenLine
}

func clear() {
	line := "\r" + strings.Repeat(" ", lastLenLine) + "\r"
	fmt.Print(line)
	lastLenLine = 0
}
