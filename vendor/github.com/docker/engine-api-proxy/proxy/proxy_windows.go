// +build windows

package proxy

import (
	"errors"
	"log"
	"net"
)

func newListener(host, groupname string) (net.Listener, error) {
	log.Fatalln("NOT IMPLEMENTED FOR THIS PLATFORM")
	return nil, errors.New("NOT IMPLEMENTED FOR THIS PLATFORM")
}
