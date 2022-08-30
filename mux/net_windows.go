//go:build windows

package mux

import (
	"context"
	"golang.org/x/sys/windows"
	"log"
	"net"
	"syscall"
)

func Listen(context context.Context, network, address string) (net.Listener, error) {
	var lc = net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				err := windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
				if err != nil {
					log.Fatal(err)
				}
			})
		},
	}
	return lc.Listen(context, network, address)
}
