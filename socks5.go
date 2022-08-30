package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net"
)

func (s *SocksServer) handleAuth(con net.Conn) error {
	buf := make([]byte, 256)
	n, err := io.ReadFull(con, buf[:1])
	if n != 1 {
		return errors.New("read header :err" + err.Error())
	}
	nmethods := int(buf[0])
	n, err = io.ReadFull(con, buf[:nmethods])
	if n != nmethods {
		return errors.New("read methods error:" + err.Error())
	}

	n, err = con.Write([]byte{0x05, 0x00})
	if n != 2 || err != nil {
		return errors.New("write auth response error:" + err.Error())
	}
	return nil
}

/**

  The SOCKS request is formed as follows:

       +----+-----+-------+------+----------+----------+
       |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
       +----+-----+-------+------+----------+----------+
       | 1  |  1  | X'00' |  1   | Variable |    2     |
       +----+-----+-------+------+----------+----------+

    Where:

         o  VER    protocol version: X'05'
         o  CMD
            o  CONNECT X'01'
            o  BIND X'02'
            o  UDP ASSOCIATE X'03'
         o  RSV    RESERVED
         o  ATYP   address type of following address
            o  IP V4 address: X'01'
            o  DOMAINNAME: X'03'
            o  IP V6 address: X'04'
         o  DST.ADDR       desired destination address
         o  DST.PORT desired destination port in network octet
            order
*/

func (s *SocksServer) handleSocks5(con net.Conn) error {
	buf := make([]byte, 256)
	n, err := io.ReadFull(con, buf[:3])
	if n != 3 {
		return errors.New("read connect header :err" + err.Error())
	}
	ver, cmd := int(buf[0]), int(buf[1])
	if ver != 5 {
		return errors.New("bad version")
	}
	addr := ""
	n, err = io.ReadFull(con, buf[:1])
	atype := buf[0]
	if atype == ATYPE_IPV4 {
		n, err = io.ReadFull(con, buf[:4])
		addr = net.IP(buf[:4]).String()
	} else if atype == ATYPE_DOMAINNAME {
		n, err = io.ReadFull(con, buf[:1])
		addrLen := int(buf[0])
		n, err = io.ReadFull(con, buf[:addrLen])
		addr = string(buf[:addrLen])
	} else if atype == ATYPE_IPV6 {
		n, err = io.ReadFull(con, buf[:16])
		addr = string('[') + (net.IP(buf[:16]).String()) + string(']')
		logrus.Infoln("ipv6:" + addr)
	}

	n, err = io.ReadFull(con, buf[:2])
	port := binary.BigEndian.Uint16(buf[:2])
	if cmd == CMD_CONNECT {
		return s.handleConnectCmd(con, addr, port)
	} else if cmd == CMD_UDP {
		return s.handleUdpCmd(con, addr, port)
	} else {
		return errors.New("not support cmd")
	}
}

func (s *SocksServer) handleConnectCmd(con net.Conn, addr string, port uint16) error {
	destAddrPort := fmt.Sprintf("%s:%d", addr, port)
	dest, err := net.Dial("tcp", destAddrPort)

	/**
	  The SOCKS request information is sent by the client as soon as it has
	     established a connection to the SOCKS server, and completed the
	     authentication negotiations.  The server evaluates the request, and
	     returns a reply formed as follows:

	          +----+-----+-------+------+----------+----------+
	          |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	          +----+-----+-------+------+----------+----------+
	          | 1  |  1  | X'00' |  1   | Variable |    2     |
	          +----+-----+-------+------+----------+----------+

	       Where:

	            o  VER    protocol version: X'05'
	            o  REP    Reply field:
	               o  X'00' succeeded
	               o  X'01' general SOCKS server failure
	               o  X'02' connection not allowed by ruleset
	               o  X'03' Network unreachable
	               o  X'04' Host unreachable
	               o  X'05' Connection refused
	               o  X'06' TTL expired
	               o  X'07' Command not supported
	               o  X'08' Address type not supported
	               o  X'09' to X'FF' unassigned
	            o  RSV    RESERVED
	            o  ATYP   address type of following address
	               o  IP V4 address: X'01'
	               o  DOMAINNAME: X'03'
	               o  IP V6 address: X'04'
	            o  BND.ADDR       server bound address
	            o  BND.PORT       server bound port in network octet order

	     Fields marked RESERVED (RSV) must be set to X'00'.
	*/

	if err != nil {
		_, _err := con.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		if _err != nil {
			logrus.Errorln(err)
			return err
		}
		return errors.New("connect dist error :" + err.Error())
	}

	_, err = con.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	if err != nil {
		return errors.New("write  response error:" + err.Error())
	}

	forward := func(src net.Conn, dest net.Conn) {
		defer func(src, dest net.Conn) {
			_ = dest.Close()
			_ = src.Close()
		}(src, dest)
		_, _ = io.Copy(dest, src)
	}
	logrus.Infoln(con.RemoteAddr().String() + "<->" + dest.LocalAddr().String() + "-" + dest.RemoteAddr().String() + " connect established!")
	go forward(con, dest)
	go forward(dest, con)
	return nil
}

func (s *SocksServer) handleUdpCmd(con net.Conn, addr string, port uint16) error {
	logrus.Infof("udp ASSOCIATE request %s:%d\n", addr, port)
	/**
	  The SOCKS request information is sent by the client as soon as it has
	     established a connection to the SOCKS server, and completed the
	     authentication negotiations.  The server evaluates the request, and
	     returns a reply formed as follows:

	          +----+-----+-------+------+----------+----------+
	          |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	          +----+-----+-------+------+----------+----------+
	          | 1  |  1  | X'00' |  1   | Variable |    2     |
	          +----+-----+-------+------+----------+----------+

	       Where:

	            o  VER    protocol version: X'05'
	            o  REP    Reply field:
	               o  X'00' succeeded
	               o  X'01' general SOCKS server failure
	               o  X'02' connection not allowed by ruleset
	               o  X'03' Network unreachable
	               o  X'04' Host unreachable
	               o  X'05' Connection refused
	               o  X'06' TTL expired
	               o  X'07' Command not supported
	               o  X'08' Address type not supported
	               o  X'09' to X'FF' unassigned
	            o  RSV    RESERVED
	            o  ATYP   address type of following address
	               o  IP V4 address: X'01'
	               o  DOMAINNAME: X'03'
	               o  IP V6 address: X'04'
	            o  BND.ADDR       server bound address
	            o  BND.PORT       server bound port in network octet order

	     Fields marked RESERVED (RSV) must be set to X'00'.

	  The UDP ASSOCIATE request is used to establish an association within
	     the UDP relay process to handle UDP datagrams.  The DST.ADDR and
	     DST.PORT fields contain the address and port that the client expects
	     to use to send UDP datagrams on for the association.  The server MAY
	     use this information to limit access to the association.  If the
	     client is not in possesion of the information at the time of the UDP
	     ASSOCIATE, the client MUST use a port number and address of all
	     zeros.

	     A UDP association terminates when the TCP connection that the UDP
	     ASSOCIATE request arrived on terminates.

	     In the reply to a UDP ASSOCIATE request, the BND.PORT and BND.ADDR
	     fields indicate the port number/address where the client MUST send
	     UDP request messages to be relayed.
	*/
	udpAddr, _ := net.ResolveIPAddr("ip", s.udpIp)
	hostByte := udpAddr.IP.To4()
	portByte := make([]byte, 2)
	binary.BigEndian.PutUint16(portByte, uint16(s.udpPort))
	buf := append([]byte{0x05, 0x00, 0x00, 0x01}, hostByte...)
	buf = append(buf, portByte...)
	_, err := con.Write(buf)
	//_,err := con.Write([]byte{0x05,0x00,0x00,0x01,0x0a,0x14,0xb,0x71,0x0f,0xa0})
	if err != nil {
		return errors.New("write response error:" + err.Error())
	}

	forward := func(src net.Conn) {
		defer func(src net.Conn) {
			_ = src.Close()
		}(src)
		for {
			_, err := io.ReadFull(src, make([]byte, 100))
			if err != nil {
				break
			}
		}
	}

	go forward(con)
	return nil
}
