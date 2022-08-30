//go:build linux

package mux

import (
	"context"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"syscall"
)

func Listen(context context.Context, network, address string) (net.Listener, error) {
	var lc = net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEADDR, 1)
				if err != nil {
					log.Fatal(err)
				}
				err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
				if err != nil {
					log.Fatal(err)
				}
			})
		},
	}
	return lc.Listen(context, network, address)
}
