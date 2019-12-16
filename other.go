//
// +build !linux, !freebsd
//

package netpoll

import "time"
import "errors"

func New(closed_ttl time.Duration) (self * Netpoll_t, err error) {
	return nil, errors.New("NOT IMPLEMENTED")
}

func (self * Netpoll_t) Listen(ip string, port int, zone uint32, backlog int) (err error) {
	return errors.New("NOT IMPLEMENTED")
}

func (self * Netpoll_t) Add(fd int) (err error) {
	return errors.New("NOT IMPLEMENTED")
}

func (self * Netpoll_t) Del(fd int) (err error) {
	return errors.New("NOT IMPLEMENTED")
}

func (self * Netpoll_t) Wait(events_size int) (err error) {
	return errors.New("NOT IMPLEMENTED")
}

func (self * Netpoll_t) Stop() (err error) {
	return errors.New("NOT IMPLEMENTED")
}

func (self * Netpoll_t) Close() (err error) {
	return errors.New("NOT IMPLEMENTED")
}
