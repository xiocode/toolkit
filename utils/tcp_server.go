/**
 * Author:        Tony.Shao
 * Email:         xiocode@gmail.com
 * Github:        github.com/xiocode
 * File:          tcp_server.go
 * Description:   Tcp Server
 */

package utils

import (
	"log"
	"net"
	"runtime"
	"strings"
)

type TcpHandler interface {
	Handle(net.Conn)
}

func TcpServer(listener net.Listener, handler TcpHandler) {
	log.Printf("TCP: listening on %s", listener.Addr().String())

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				log.Printf("NOTICE: temporary Accept() failure - %s", err.Error())
				runtime.Gosched()
				continue
			}
			// theres no direct way to detect this error because it is not exposed
			if !strings.Contains(err.Error(), "use of closed network connection") {
				log.Printf("ERROR: listener.Accept() - %s", err.Error())
			}
			break
		}
		go handler.Handle(clientConn)
	}

	log.Printf("TCP: closing %s", listener.Addr().String())
}
