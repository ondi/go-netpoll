//
// +build !linux,!freebsd
//

package netpoll

import (
	"errors"
	"time"
)

func New(closed_ttl time.Duration) (*Netpoll_t, error) {
	return nil, errors.New("NOT IMPLEMENTED")
}

func (self *Netpoll_t) Listen(ip string, port int, zone uint32, backlog int) error {
	return errors.New("NOT IMPLEMENTED")
}

func (self *Netpoll_t) Add(fd int) error {
	return errors.New("NOT IMPLEMENTED")
}

func (self *Netpoll_t) Del(fd int) error {
	return errors.New("NOT IMPLEMENTED")
}

func (self *Netpoll_t) Wait(events_size int) error {
	return errors.New("NOT IMPLEMENTED")
}

func (self *Netpoll_t) Stop() error {
	return errors.New("NOT IMPLEMENTED")
}

func (self *Netpoll_t) Close() error {
	return errors.New("NOT IMPLEMENTED")
}

func GetIpPort(Sockaddr interface{}) (string, int) {
	return "", 0
}

func Getpeername(fd int) (string, int, error) {
	return "", 0, errors.New("NOT IMPLEMENTED")
}

func Getsockname(fd int) (string, int, error) {
	return "", 0, errors.New("NOT IMPLEMENTED")
}

type FD int

func (self FD) Read(p []byte) (int, error) {
	return 0, errors.New("NOT IMPLEMENTED")
}

func (self FD) Write(p []byte) (int, error) {
	return 0, errors.New("NOT IMPLEMENTED")
}
