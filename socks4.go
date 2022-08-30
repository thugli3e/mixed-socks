package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net"
)

func (s *SocksServer) handleSocks4(con net.Conn) error {
	buf := make([]byte, 256)
	n, err := io.ReadFull(con, buf[:1])
	if n != 1 {
		return errors.New("read header :err" + err.Error())
	}
	cmd := int(buf[0])

	n, err = io.ReadFull(con, buf[:2])
	port := binary.BigEndian.Uint16(buf[:2])

	n, err = io.ReadFull(con, buf[:4])
	addr := net.IP(buf[:4]).String()

	/**
	    IP address 0.0.0.x, with x nonzero,
	  an inadmissible destination address and
	  thus should never occur if the client can resolve the domain name.)
	  Following the NULL byte terminating USERID,
	  the client must send the destination domain name
	  and terminate it with another NULL byte.
	  This is used for both "connect" and "bind" requests.
	*/
	var useDomain = false
	if buf[0] == 0x00 && buf[1] == 0x00 && buf[2] == 0x00 && buf[3] != 0x00 {
		useDomain = true
	}

	for {
		n, err = io.ReadFull(con, buf[:1])
		if err != nil {
			return errors.New("read userid error :" + err.Error())
		}
		if buf[0] == 0x00 {
			break
		}
	}
	if useDomain {
		var i = 0
		for {
			n, err = io.ReadFull(con, buf[i:i+1])
			if err != nil {
				return errors.New("read userid error :" + err.Error())
			}
			if buf[i] == 0x00 {
				break
			}
			i++
		}
		addr = string(buf[:i])
	}

	if cmd == CMD_CONNECT {
		return s.handleSock4ConnectCmd(con, addr, port)
	} else {
		return errors.New("not support cmd")
	}
}

func (s *SocksServer) handleSock4ConnectCmd(con net.Conn, addr string, port uint16) error {

	destAddrPort := fmt.Sprintf("%s:%d", addr, port)
	dest, err := net.Dial("tcp", destAddrPort)

	/**
	  The SOCKS server uses the client information to decide whether the
	  request is to be granted. The reply it sends back to the client has
	  the same format as the reply for CONNECT request, i.e.,

	  		+----+----+----+----+----+----+----+----+
	  		| VN | CD | DSTPORT |      DSTIP        |
	  		+----+----+----+----+----+----+----+----+
	  # of bytes:	   1    1      2              4

	  	VN
	      reply version, null byte
	  REP
	      reply code

	      Byte 	Meaning
	      0x5A 	Request granted
	      0x5B 	Request rejected or failed
	      0x5C 	Request failed because client is not running identd (or not reachable from server)
	      0x5D 	Request failed because client's identd could not confirm the user ID in the request

	  DSTPORT
	      destination port, meaningful if granted in BIND, otherwise ignore
	  DSTIP
	      destination IP, as above – the ip:port the client should bind to
	*/

	if err != nil {
		_, _err := con.Write([]byte{0x00, 0x5B, 0x00, 0x00, 0, 0, 0, 0})
		if _err != nil {
			return err
		}
		return errors.New("connect dist error :" + err.Error())
	}

	_, err = con.Write([]byte{0x00, 0x5A, 0x00, 0x00, 0, 0, 0, 0})
	if err != nil {
		return errors.New("write  response error:" + err.Error())
	}

	forward := func(src net.Conn, dest net.Conn) {
		defer src.Close()
		defer dest.Close()
		_, err := io.Copy(dest, src)
		if err != nil {
			logrus.Infoln(src.RemoteAddr().String(), err)
		}
	}
	logrus.Infoln(con.RemoteAddr().String() + "-" + dest.LocalAddr().String() + "-" + dest.RemoteAddr().String() + " connect established!")
	go forward(con, dest)
	go forward(dest, con)
	return nil
}
