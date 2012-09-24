package build

import (
	"os"
	"sync"
	"time"
)

// TODO test CopyFile
type CopyFile struct {
	res    *ResourceManager
	source Node
	name   string
	cond   *sync.Cond
	conns  uint // number of actual connections
	conned bool // no more connections will arrive
	stat   *StatInfo
	read   *ReadInfo
	clean  bool // cleanup done
}

func NewCopyFile(res *ResourceManager, source Node, name string) *CopyFile {
	source.Connect()
	return &CopyFile{res, source, name, sync.NewCond(&sync.Mutex{}), 0, false, nil, nil, false}
}

func (c *CopyFile) Connect() {
	c.cond.L.Lock()
	c.conns++
	c.cond.L.Unlock()
}

func (c *CopyFile) Disconnect() {
	c.cond.L.Lock()
	c.conns--
	c.cond.L.Unlock()
	go c.cleanup()
}

func (c *CopyFile) Connected() {
	c.cond.L.Lock()
	c.conned = true
	c.cond.L.Unlock()
	go c.cleanup()
}

func (c *CopyFile) Work() error {
	c.res.Start()

	sourceStat := c.source.Stat()
	if sourceStat.err != nil {
		c.cond.L.Lock()
		c.stat, c.read = &StatInfo{time.Time{}, 0, sourceStat.err}, &ReadInfo{nil, sourceStat.err}
		c.cond.Broadcast()
		c.cond.L.Unlock()
		c.source.Disconnect()
		c.res.Finish()
		go c.cleanup()
		return sourceStat.err
	}

	var (
		targetStat *StatInfo
		outofdate  bool
	)
	if stat, err := c.res.Stat(c.name); err != nil {
		if os.IsNotExist(err) {
			outofdate = true
		} else {
			c.cond.L.Lock()
			c.stat, c.read = &StatInfo{time.Time{}, 0, err}, &ReadInfo{nil, err}
			c.cond.Broadcast()
			c.cond.L.Unlock()
			c.source.Disconnect()
			c.res.Finish()
			go c.cleanup()
			return err
		}
	} else {
		targetStat = &StatInfo{stat.ModTime(), stat.Size(), nil}
		outofdate = targetStat.size != sourceStat.size || targetStat.time.Before(sourceStat.time)
	}

	var buf *Buffer
	if outofdate {

		read := c.source.Read()
		if read.err != nil {
			c.cond.L.Lock()
			c.stat, c.read = &StatInfo{time.Time{}, 0, read.err}, &ReadInfo{nil, read.err}
			c.cond.Broadcast()
			c.cond.L.Unlock()
			c.source.Disconnect()
			c.res.Finish()
			go c.cleanup()
			return read.err
		}

		buf = read.buf
		err := c.res.Write(c.name, buf)
		if err != nil {
			read.buf.DecrRef()
			c.cond.L.Lock()
			c.stat, c.read = &StatInfo{time.Time{}, 0, err}, &ReadInfo{nil, err}
			c.cond.Broadcast()
			c.cond.L.Unlock()
			c.source.Disconnect()
			c.res.Finish()
			go c.cleanup()
			return err
		}

		targetStat = &StatInfo{time.Now(), int64(len(buf.Data())), nil}
	}

	c.cond.L.Lock()
	c.stat, c.read = targetStat, &ReadInfo{buf, nil}
	c.cond.Broadcast()
	c.cond.L.Unlock()
	c.source.Disconnect()
	c.res.Finish()
	go c.cleanup()

	return nil
}

func (c *CopyFile) Stat() StatInfo {
	c.cond.L.Lock()
	for c.stat == nil {
		c.cond.Wait()
	}
	s := *c.stat
	c.cond.L.Unlock()
	return s
}

func (c *CopyFile) Read() ReadInfo {
	c.cond.L.Lock()
	for c.read == nil {
		c.cond.Wait()
	}
	if c.read.buf == nil && c.read.err == nil {
		// TODO while reading here, one client could be locked on Stat()
		c.read.buf, c.read.err = c.res.Read(c.name, c.stat.size)
	}
	r := *c.read
	c.cond.L.Unlock()
	if r.buf != nil {
		r.buf.IncrRef()
	}
	return r
}

func (c *CopyFile) cleanup() {
	c.cond.L.Lock()
	if !c.clean && c.conned && c.conns == 0 && c.stat != nil {
		c.source = nil
		c.stat.err = nil
		c.stat = nil
		if c.read.buf != nil {
			c.read.buf.DecrRef()
			c.read.buf = nil
		}
		c.read.err = nil
		c.read = nil
		c.clean = true
	}
	c.cond.L.Unlock()
}
