package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/antonovegorv/simple-tracepath-ipv4/tracepath"
	"github.com/antonovegorv/simple-tracepath-ipv4/tracepath/config"
)

const numberOfTracers = 1

func main() {
	timeout := flag.Int("i", 4, "max timeout for a reply")
	maxTTL := flag.Int("t", 64, "max number of hops")
	packetSize := flag.Int("s", 64, "size of a single packet in bytes")
	flag.Parse()

	var hostname string
	if hostname = flag.Arg(0); hostname == "" {
		fmt.Println("You have to provide hostname to trace with")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	wg := &sync.WaitGroup{}
	wg.Add(numberOfTracers)

	errorsChan := make(chan error, 1)

	t := tracepath.New(ctx, wg, errorsChan, config.New(
		hostname,
		*timeout,
		*maxTTL,
		*packetSize,
	))
	go t.Trace()

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

Loop:
	for {
		select {
		case <-termChan:
			cancel()
			break Loop
		case err := <-errorsChan:
			if err != nil {
				fmt.Println(err)
			}
			break Loop
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	wg.Wait()
}
