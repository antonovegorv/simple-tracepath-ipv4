package tracepath

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/antonovegorv/simple-tracepath-ipv4/tracepath/config"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const network = "ip4:icmp"
const address = "0.0.0.0"
const protocolICMP = 1

type Tracepath struct {
	ctx        context.Context
	wg         *sync.WaitGroup
	errorsChan chan error
	config     *config.Config
}

func New(ctx context.Context, wg *sync.WaitGroup, errorsChan chan error,
	config *config.Config) *Tracepath {
	return &Tracepath{
		ctx:        ctx,
		wg:         wg,
		errorsChan: errorsChan,
		config:     config,
	}
}

func (t *Tracepath) Trace() {
	defer t.wg.Done()

	c, err := icmp.ListenPacket(network, address)
	if err != nil {
		t.errorsChan <- err
		return
	}
	defer c.Close()

	ips, err := net.LookupIP(t.config.Hostname)
	if err != nil {
		t.errorsChan <- err
		return
	}

	var destIP net.IP
	for _, ip := range ips {
		if destIP = ip.To4(); destIP != nil {
			break
		}
	}

	if destIP == nil {
		t.errorsChan <- fmt.Errorf("no ipv4 for that host %v;", t.config.Hostname)
		return
	}

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte(strings.Repeat("0", t.config.PacketSize)),
		},
	}

	wb, err := wm.Marshal(nil)
	if err != nil {
		t.errorsChan <- err
		return
	}

	rb := make([]byte, 1500)

	for i := 1; i <= t.config.MaxTTL; i++ {
		select {
		case <-t.ctx.Done():
			return
		default:
			c.IPv4PacketConn().SetTTL(i)

			start := time.Now()

			if _, err := c.WriteTo(wb, &net.IPAddr{IP: destIP}); err != nil {
				t.errorsChan <- err
				return
			}

			err = c.SetReadDeadline(time.Now().Add(time.Duration(t.config.Timeout) * time.Second))
			if err != nil {
				t.errorsChan <- err
				return
			}

			n, peer, err := c.ReadFrom(rb)
			if err != nil {
				fmt.Printf("%2d: no reply\n", i)
				continue
			}

			elapsed := time.Since(start)

			rm, err := icmp.ParseMessage(protocolICMP, rb[:n])
			if err != nil {
				t.errorsChan <- err
				return
			}

			switch rm.Type {
			case ipv4.ICMPTypeTimeExceeded:
				fmt.Printf("%2d: %-64v %v\n", i, getDomain(peer), elapsed)
			case ipv4.ICMPTypeEchoReply:
				fmt.Printf("%2d: %-64v %v\n", i, getDomain(peer), elapsed)
				t.errorsChan <- nil
				return
			}
		}
	}
}

func getDomain(peer net.Addr) string {
	host, err := net.LookupAddr(peer.String())
	if err != nil {
		return peer.String()
	}
	return host[0]
}
