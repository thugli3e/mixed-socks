package proxy

import (
	"context"
	"github.com/sirupsen/logrus"
)

const (
	CMD_CONNECT      = 0x01
	CMD_BIND         = 0x02
	CMD_UDP          = 0x03
	ATYPE_IPV4       = 0x01
	ATYPE_DOMAINNAME = 0x03
	ATYPE_IPV6       = 0x04
)

type SocksServer struct {
	sockIp  string
	port    int
	udpIp   string // udp associate ip
	udpPort int    // udp associate address
}

func NewSocksServer(host string, port int) *SocksServer {
	socksServer := SocksServer{
		sockIp:  host,
		port:    port,
		udpIp:   host,
		udpPort: port,
	}
	return &socksServer
}

// ListenAndServe socks4 socks5 server
func (s *SocksServer) ListenAndServe(ctx context.Context) {
	go func() {
		udpServer := NewUdpServer()
		err := udpServer.Listen()
		if err != nil {
			logrus.Fatalln(err)
		}
		s.udpIp = udpServer.udpIp
		s.udpPort = udpServer.udpPort
	}()
	err := s.listenTcpServer(ctx)
	if err != nil {
		logrus.Fatalln(err)
	}
}
