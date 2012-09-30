package build

import (
	"sync"
	"time"
)

type SourceFile struct {
	res    *ResourceManager
	name   string
	connx  sync.Mutex // access to conns and conned
	conns  uint       // number of actual connections
	conned bool       // no more connections will arrive
	statx  sync.Mutex // access to stat
	stat   *StatInfo
	readx  sync.Mutex // access to read
	read   *ReadInfo
}

func NewSourceFile(r *ResourceManager, name string) *SourceFile {
	return &SourceFile{r, name, sync.Mutex{}, 0, false, sync.Mutex{}, nil, sync.Mutex{}, nil}
}

func (s *SourceFile) Name() string {
	return s.name
}

func (s *SourceFile) Connect() {
	s.connx.Lock()
	s.conns++
	s.connx.Unlock()
}

func (s *SourceFile) Disconnect() {
	s.connx.Lock()
	s.conns--
	s.connx.Unlock()
	go s.cleanup()
}

func (s *SourceFile) Connected() {
	s.connx.Lock()
	s.conned = true
	s.connx.Unlock()
	go s.cleanup()
}

func (s *SourceFile) Stat() StatInfo {
	s.statx.Lock()
	if s.stat == nil {
		if stat, err := s.res.Stat(s.name); err != nil {
			s.stat = &StatInfo{time.Time{}, 0, err}
		} else {
			s.stat = &StatInfo{stat.ModTime(), stat.Size(), nil}
		}
	}
	t := *s.stat
	s.statx.Unlock()
	return t
}

func (s *SourceFile) Read() ReadInfo {
	s.readx.Lock()
	if s.read == nil {
		if stat := s.Stat(); stat.err != nil {
			s.read = &ReadInfo{nil, stat.err}
		} else {
			buf, err := s.res.Read(s.name, stat.size)
			s.read = &ReadInfo{buf, err}
		}
	}
	r := *s.read
	s.readx.Unlock()
	if r.buf != nil {
		r.buf.IncrRef()
	}
	return r
}

func (s *SourceFile) cleanup() {
	s.readx.Lock()
	s.statx.Lock()
	s.connx.Lock()
	if s.conned && s.conns == 0 {
		if s.stat != nil {
			s.stat.err = nil
			s.stat = nil
		}
		if s.read != nil {
			if s.read.buf != nil {
				s.read.buf.DecrRef()
				s.read.buf = nil
			}
			s.read.err = nil
			s.read = nil
		}
	}
	s.connx.Unlock()
	s.statx.Unlock()
	s.readx.Unlock()
}
