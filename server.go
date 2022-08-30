package proxy

import (
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"net"
	"strconv"
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
	tcpAddr *net.TCPAddr
	sockIp  string
	port    int
	udpIp   string //udp associate ip
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
func (s *SocksServer) ListenAndServe() {
	go func() {
		udpServer := NewUdpServer(s.udpIp, s.udpPort)
		err := udpServer.Listen()
		if err != nil {
			logrus.Fatalln(err)
		}
	}()
	err := s.listenTcpServer()
	if err != nil {
		logrus.Fatalln(err)
	}
}

func (s *SocksServer) listenTcpServer() error {
	s.tcpAddr, _ = net.ResolveTCPAddr("tcp", s.sockIp+":"+strconv.Itoa(s.port))
	conn, err := net.ListenTCP("tcp", s.tcpAddr)
	if err != nil {
		logrus.Infoln("connect error", err)
		return err
	}
	logrus.Infoln("Listen tcp:" + s.tcpAddr.String())

	for {
		c, err := conn.Accept()
		if err != nil {
			logrus.Errorln("accept error", err)
			break
		}
		go s.handleConnection(c)

	}
	defer func(conn *net.TCPListener) {
		err := conn.Close()
		if err != nil {
			logrus.Error(err)
		}
	}(conn)
	return errors.New("socks server stop")
}

func (s *SocksServer) handleConnection(con net.Conn) {
	logrus.Infoln(con.RemoteAddr().String() + " request for service!")
	ver, err := s.handleVersion(con)
	if err != nil {
		con.Close()
		log.Println(con.RemoteAddr().String()+" error", err)
		return
	}
	if ver == 4 {
		err := s.handleSocks4(con)
		if err != nil {
			logrus.Warningln(err)
		}
		return
	}
	if ver == 5 {
		err = s.handleAuth(con)
		if err != nil {
			con.Close()
			logrus.Warningln(con.RemoteAddr().String()+" error", err)
			return
		}

		err = s.handleSocks5(con)
		if err != nil {
			logrus.Warningln(con.RemoteAddr().String()+" error", err)
			con.Close()
		}
		return
	}
	//default handle http
	err = s.handleProxy(con, ver)
	if err != nil {
		logrus.Warningln(con.RemoteAddr().String()+" http proxy error", err)
		con.Close()
	}
	return
}

// http CONNECT first char is C
// CONNECT streamline.t-mobile.com:443 HTTP/1.1
func (s *SocksServer) handleVersion(con net.Conn) (byte, error) {

	buf := make([]byte, 1)
	n, err := io.ReadFull(con, buf[:1])
	if n != 1 {
		return 0, errors.New("read header :err" + err.Error())
	}
	ver := buf[0]
	return ver, nil
}
