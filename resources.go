package build

import (
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
)

type ResourceManager struct {
	cond                          *sync.Cond
	muse, mmax                    int64 // memory
	dstats, dreads, dwrites, dmax uint  // disk
	twait, tnum                   uint  // tasks
}

func NewResourceManager(mem int64, disk uint) *ResourceManager {
	return &ResourceManager{sync.NewCond(&sync.Mutex{}), 0, mem, 0, 0, 0, disk, 0, 0}
}

type Buffer struct {
	res   *ResourceManager
	datax sync.Mutex
	data  []byte
	refs  int64
}

func (b *Buffer) Data() []byte {
	b.datax.Lock()
	d := b.data
	b.datax.Unlock()
	return d
}

func (b *Buffer) IncrRef() int64 {
	return atomic.AddInt64(&b.refs, 1)
}

func (b *Buffer) DecrRef() int64 {
	n := atomic.AddInt64(&b.refs, -1)
	if n <= 0 {
		b.datax.Lock()
		if b.data != nil {
			b.res.cond.L.Lock()
			b.res.muse -= int64(len(b.data))
			b.res.cond.Broadcast()
			b.res.cond.L.Unlock()
			b.data = nil
		}
		b.datax.Unlock()
	}
	return n
}

func (r *ResourceManager) Start() {
	r.cond.L.Lock()
	r.tnum++
	r.cond.L.Unlock()
}

func (r *ResourceManager) Stat(name string) (os.FileInfo, error) {

	r.cond.L.Lock()
	for r.dstats+r.dreads+r.dwrites >= r.dmax {
		r.cond.Wait()
	}
	r.dstats++
	r.cond.L.Unlock()

	stat, err := os.Stat(name)

	r.cond.L.Lock()
	r.dstats--
	r.cond.Broadcast()
	r.cond.L.Unlock()

	return stat, err
}

func (r *ResourceManager) Read(name string, size int64) (*Buffer, error) {

	r.cond.L.Lock()
	for r.twait+1 < r.tnum {
		if r.dstats+r.dreads+r.dwrites >= r.dmax {
			r.cond.Wait()
		} else if r.muse+size > r.mmax {
			r.twait++
			r.cond.Wait()
			r.twait--
		} else {
			break
		}
	}
	r.muse += size
	r.dreads++
	r.cond.L.Unlock()

	data, err := ioutil.ReadFile(name)

	r.cond.L.Lock()
	var buf *Buffer
	if err != nil {
		r.muse -= size
	} else {
		r.muse += int64(len(data)) - size
		buf = &Buffer{r, sync.Mutex{}, data, 1}
	}
	r.dreads--
	r.cond.Broadcast()
	r.cond.L.Unlock()

	return buf, err
}

func (r *ResourceManager) Write(name string, buf *Buffer) error {

	r.cond.L.Lock()
	for r.dstats+r.dreads+r.dwrites >= r.dmax {
		r.cond.Wait()
	}
	r.dwrites++
	r.cond.L.Unlock()

	err := ioutil.WriteFile(name, buf.Data(), 0666)

	r.cond.L.Lock()
	r.dwrites--
	r.cond.Broadcast()
	r.cond.L.Unlock()

	return err
}

func (r *ResourceManager) Wait(cond *sync.Cond) {
	// TODO Wait()
}

func (r *ResourceManager) Finish() {
	r.cond.L.Lock()
	r.tnum--
	r.cond.Broadcast()
	r.cond.L.Unlock()
}

func (r *ResourceManager) Mem() (int64, int64) {
	r.cond.L.Lock()
	use, max := r.muse, r.mmax
	r.cond.L.Unlock()
	return use, max
}

func (r *ResourceManager) Disk() (uint, uint, uint, uint) {
	r.cond.L.Lock()
	stats, reads, writes, max := r.dstats, r.dreads, r.dwrites, r.dmax
	r.cond.L.Unlock()
	return stats, reads, writes, max
}

func (r *ResourceManager) Tasks() (uint, uint) {
	r.cond.L.Lock()
	wait, num := r.twait, r.tnum
	r.cond.L.Unlock()
	return wait, num
}
