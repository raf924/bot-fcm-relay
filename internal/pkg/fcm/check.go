package fcm

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

type SRVHost struct {
	Host string
	Port int
}

func LookupPortFromDNS(serverName string) ([]SRVHost, error) {
	_, addrs, err := net.LookupSRV("", "", fmt.Sprintf("_hc._grpc.%s", serverName))
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no matching SRV records")
	}
	var hosts []SRVHost
	for _, addr := range addrs {
		hosts = append(hosts, SRVHost{
			Host: strings.TrimRight(addr.Target, "."),
			Port: int(addr.Port),
		})
	}
	return hosts, nil
}

func CheckPairing(l net.Listener, config fcmRelayConfig) error {
	hosts, err := LookupPortFromDNS(config.ServerName)
	if err != nil {
		println("Warning:", config.ServerName, "doesn't have the required SRV record. This relay will have to be added manually to the app")
		return nil
	}
	for _, host := range hosts {
		if err := CheckConnection(l, host.Host, host.Port); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no SRV records match this relay")
}

var checkTimeout = time.Second * 20

func CheckConnection(l net.Listener, host string, port int) error {
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
