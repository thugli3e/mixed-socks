package proxy

import (
    "bufio"
    "errors"
    "fmt"
    "github.com/sirupsen/logrus"
    "io"
    "net"
    "strconv"
    "strings"
)

func readString(conn net.Conn, delim byte) (string, error) {
    buf := make([]byte, 1024)
    i := 0
    for {
        current := i
        _, err := conn.Read(buf[current : current+1])
        i++
        if err != nil {
            logrus.Warningln(err.Error())
            break
        }
        if buf[current] == delim {
            break
        }
        if i == len(buf) {
            break
        }
    }
    return string(buf[:i]), nil
}

func (s *SocksServer) handleProxy(con net.Conn, firstc byte) error {
    line, err := readString(con, '\n')
    if err != nil {
        return err
    }
    line = string(firstc) + line
    logrus.Infoln("http proxy requestLine " + strings.ReplaceAll(line, "\r\n", ""))
    requestLine := strings.Split(line, " ")
    if len(requestLine) < 3 {
        return errors.New("request line error")
    }
    method := requestLine[0]
    requestTarget := requestLine[1]
    version := requestLine[2]

    if method == "CONNECT" {
        reader := bufio.NewReader(con)
        shp := strings.Split(requestTarget, ":")
        addr := shp[0]
        port, _ := strconv.Atoi(shp[1])
        //consume rest header
        for {
            line, err = reader.ReadString('\n')
            if line == "\r\n" {
                break
            }
            logrus.Infoln("rest:" + strings.ReplaceAll(line, "\r\n", ""))
        }
        return s.handleHTTPConnectMethod(con, addr, uint16(port))
    } else {
        si := strings.Index(requestTarget, "//")
        restUrl := requestTarget[si+2:]
        port := 80
        ei := strings.Index(restUrl, "/")
        url := "/"
        hostPort := restUrl
        if ei != -1 {
            hostPort = restUrl[:ei]
            url = restUrl[ei:]
        }
        as := strings.Split(hostPort, ":")
        addr := as[0]
        if len(as) == 2 {
            port, _ = strconv.Atoi(as[1])
        }

        newline := method + " " + url + " " + version
        logrus.Infoln("http proxy newline " + newline)
        return s.handleHTTPProxy(con, addr, uint16(port), newline)
    }
}

func (s *SocksServer) handleHTTPConnectMethod(con net.Conn, addr string, port uint16) error {

    destAddrPort := fmt.Sprintf("%s:%d", addr, port)
    dest, err := net.Dial("tcp", destAddrPort)
    /**

     */
    if err != nil {
        //con.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
        return errors.New("connect dist error :" + err.Error())
    }
    _, err = con.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

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

// 后续的request line都是全路径，某些服务器可能有问题

func (s *SocksServer) handleHTTPProxy(con net.Conn, addr string, port uint16, line string) error {

    destAddrPort := fmt.Sprintf("%s:%d", addr, port)
    dest, err := net.Dial("tcp", destAddrPort)
    /**
     */
    if err != nil {
        return errors.New("connect dist error :" + err.Error())
    }
    _, err = dest.Write([]byte(line))
    if err != nil {
        return errors.New("write  response error:" + err.Error())
    }
    forward := func(src net.Conn, dest net.Conn) {
        defer src.Close()
        defer dest.Close()
        _, err := io.Copy(dest, src)
        if err != nil {
            logrus.Warningln(src.RemoteAddr().String(), err)
        }
    }
    logrus.Infoln(con.RemoteAddr().String() + "-" + dest.LocalAddr().String() + "-" + dest.RemoteAddr().String() + " connect established!")
    go forward(con, dest)
    go forward(dest, con)
    return nil
}
