package fcm

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

func LookupPortFromDNS(serverName string) (host string, port int, err error) {
	_, addrs, err := net.LookupSRV("", "", fmt.Sprintf("_hc._grpc.%s", serverName))
	if err != nil {
		return "", 0, err
	}
	addr := addrs[0]
	return strings.TrimRight(addr.Target, "."), int(addr.Port), nil
}

func CheckPairing(config fcmRelayConfig) error {
	host, port, err := LookupPortFromDNS(config.ServerName)
	if err != nil {
		println("Warning:", config.ServerName, "doesn't have the required SRV record. This relay will have to be added manually to the app")
		return nil
	}
	return CheckConnection(host, port)
}

const checkTimeout = time.Second * 3

func CheckConnection(host string, port int) error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer l.Close()
	sameChan := make(chan bool)
	dConn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), checkTimeout)
	if err != nil {
		return fmt.Errorf("relay is not served on %s:%d", host, port)
	}
	_ = dConn.SetWriteDeadline(time.Now().Add(checkTimeout))
	defer dConn.Close()
	rand.Seed(time.Now().UnixNano())
	secret := make([]byte, 20)
	_, _ = rand.Read(secret)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			sameChan <- false
			return
		}
		defer conn.Close()
		b := make([]byte, len(secret))
		_, err = conn.Read(b)
		if err != nil {
			sameChan <- false
			return
		}
		sameChan <- bytes.Equal(secret, b)
	}()
	_, err = dConn.Write(secret)
	if err != nil {
		return fmt.Errorf("relay is not served on %s:%d", host, port)
	}
	isSame := <-sameChan
	if !isSame {
		return fmt.Errorf("relay is not served on %s:%d", host, port)
	}
	return nil
}
