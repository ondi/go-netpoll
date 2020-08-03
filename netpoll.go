//
// Repeat Read() and Write() untill EAGAIN.
// AddFd() and DelFd() should be under user fd cache mutex if exists.
// Local cache prevents fd to run on different threads,
// provides a read queue between epoll_wait() and Read()
// and short lived state.Data for slow devices which interact like so:
// WEBSOCKET_HEADER -> EAGAIN -> WEBSOCKET_BODY -> EAGAIN
//
// RACES:
// fd may be deleted, closed and reopened right after epoll_wait()
// and before processing. Read() may get fd with outdated events.
// the following race condition present in code below:
// one thread may call Del(fd), Add(fd) while events from epoll_wait()
// still processing and closed flag already unset.
//
// see 'Possible Pitfalls and Ways to Avoid Them' for details
// https://linux.die.net/man/4/epoll
//

package netpoll

import (
	"net"
	"sync"
	"time"

	"github.com/ondi/go-cache"
)

const (
	FLAG_RUNNING uint64 = 1 << 63
	FLAG_CLOSED  uint64 = 1 << 62
)

type READ func(int, *State_t)

type State_t struct {
	updated time.Time
	events  uint64
	Data    interface{}
}

type Netpoll_t struct {
	poller int
	event  int
	listen int
	ttl    time.Duration

	mx      sync.Mutex
	cond    *sync.Cond
	ready   *cache.Cache_t
	added   int
	running bool
}

func (self *Netpoll_t) __set_fd_open(fd int) {
	self.ready.UpdateBack(fd, func(value interface{}) interface{} {
		value.(*State_t).updated = time.Now()
		value.(*State_t).events &= FLAG_RUNNING
		return value
	})
}

func (self *Netpoll_t) __set_fd_closed(fd int) {
	it, ok := self.ready.PushBack(fd, func() interface{} {
		return &State_t{updated: time.Now(), events: FLAG_CLOSED}
	})
	if !ok {
		it.Value().(*State_t).events |= FLAG_CLOSED
	}
}

func (self *Netpoll_t) set_fd_closed(fd int) {
	self.mx.Lock()
	self.__set_fd_closed(fd)
	self.mx.Unlock()
}

func (self *Netpoll_t) AddEvent(fd int) {
	self.mx.Lock()
	defer self.mx.Unlock()
	it, ok := self.ready.PushBack(fd, func() interface{} { return &State_t{updated: time.Now(), events: 1} })
	if ok {
		self.cond.Signal()
		return
	}
	if it.Value().(*State_t).events&FLAG_CLOSED == FLAG_CLOSED {
		return
	}
	it.Value().(*State_t).updated = time.Now()
	it.Value().(*State_t).events++
	if it.Value().(*State_t).events&FLAG_RUNNING == 0 {
		self.cond.Signal()
	}
}

func (self *Netpoll_t) Read(fn READ) {
	self.mx.Lock()
	for self.running {
		self.cond.Wait()
	loop:
		now := time.Now()
		for i := 0; i < self.ready.Size(); {
			it := self.ready.Front()
			state := it.Value().(*State_t)
			if state.events&FLAG_RUNNING == FLAG_RUNNING {
				cache.MoveBefore(it, self.ready.End())
				i++
				continue
			}
			if state.events & ^FLAG_CLOSED == 0 {
				if now.Sub(state.updated) > self.ttl {
					self.ready.Remove(it.Key())
					continue
				}
				cache.MoveBefore(it, self.ready.End())
				i++
				continue
			}
			state.events &= FLAG_CLOSED
			state.events |= FLAG_RUNNING
			cache.MoveBefore(it, self.ready.End())
			self.mx.Unlock()
			fn(it.Key().(int), it.Value().(*State_t))
			self.mx.Lock()
			state.events &= ^FLAG_RUNNING
			goto loop
		}
	}
}

func (self *Netpoll_t) SizeAdded() int {
	self.mx.Lock()
	defer self.mx.Unlock()
	return self.added
}

func (self *Netpoll_t) SizeReady() int {
	self.mx.Lock()
	defer self.mx.Unlock()
	return self.ready.Size()
}

func GetFd(conn net.Conn) (fd int) {
	if tcp_conn, ok := conn.(*net.TCPConn); ok {
		if raw, err := tcp_conn.SyscallConn(); err == nil {
			raw.Control(
				func(c uintptr) {
					fd = int(c)
				},
			)
			return
		}
	}
	return -1
}
