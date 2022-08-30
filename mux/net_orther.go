//go:build !linux && !windows

package mux

import (
	"context"
	"net"
)

func Listen(context context.Context, network, address string) (net.Listener, error) {
	var lc net.ListenConfig
	return lc.Listen(context, network, address)
}
