package proxy

import (
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"net"
	"strconv"
)

func (s *SocksServer) listenTcpServer() error {
	s.tcpAddr, _ = net.ResolveTCPAddr("tcp", s.sockIp+":"+strconv.Itoa(s.port))
	conn, err := net.ListenTCP("tcp", s.tcpAddr)
	if err != nil {
		logrus.Infoln("connect error", err)
		return err
	}
	logrus.Infoln("listen tcp:" + s.tcpAddr.String())

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
	ver, err := s.handleVersion(con)
	if err != nil {
		_ = con.Close()
		log.Println(con.RemoteAddr().String()+" error", err)
		return
	}
	if ver == 4 {
		logrus.Infoln(con.RemoteAddr().String(), "using socks4 request for service!")
		err := s.handleSocks4(con)
		if err != nil {
			logrus.Warningln(err)
		}
		return
	}
	if ver == 5 {
		logrus.Infoln(con.RemoteAddr().String(), "using socks5 request for service!")
		err = s.handleAuth(con)
		if err != nil {
			_ = con.Close()
			logrus.Warningln(con.RemoteAddr().String()+" error", err)
			return
		}

		err = s.handleSocks5(con)
		if err != nil {
			logrus.Warningln(con.RemoteAddr().String()+" error", err)
			_ = con.Close()
		}
		return
	}
	//default handle http
	logrus.Infoln(con.RemoteAddr().String(), "using http request for service!")
	err = s.handleProxy(con, ver)
	if err != nil {
		logrus.Warningln(con.RemoteAddr().String()+" http proxy error", err)
		_ = con.Close()
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
