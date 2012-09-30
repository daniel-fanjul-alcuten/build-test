package build

import (
	"time"
)

type StatInfo struct {
	time time.Time
	size int64
	err  error
}

type ReadInfo struct {
	buf *Buffer
	err error
}

// TODO split into Node and FileNode
// TODO add Done() (bool, error)
type Node interface {
	// clients notify they start
	Connect()
	// clients notify they finish
	Disconnect()
	// no more clients will connect
	Connected()
	// output must be cached
	Stat() StatInfo
	// output must be cached
	Read() ReadInfo
}

type Worker interface {
	Work() error
}
