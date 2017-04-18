package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

const MAX_PACKETS = 256

var serverMode bool
var maxClients int
var runTime float64

var addr = &net.UDPAddr{IP: net.ParseIP("::1"), Port: 40000}

func init() {
	flag.BoolVar(&serverMode, "server", false, "set false for client mode")
	flag.IntVar(&maxClients, "num", 64, "number of clients to serve or to create")
	flag.Float64Var(&runTime, "runtime", 5.0, "how long to run clients for/clear client buffer in seconds")

}

func main() {
	flag.Parse()
	packetData := make([]byte, 1220)
	for i := 0; i < len(packetData); i += 1 {
		packetData[i] = byte(i)
	}

	if serverMode {
		serveLoop(packetData)
		return
	}
	wg := &sync.WaitGroup{}
	log.Printf("starting %d clients\n", maxClients)
	for i := 0; i < maxClients; i += 1 {
		wg.Add(1)
		go clientLoop(wg, packetData, i)
	}
	wg.Wait()
}

func serveLoop(packetData []byte) {
	serverTime := float64(0)
	delta := float64(1.0 / 60.0)
	deltaTime := time.Duration(delta * float64(time.Second))
	count := 0

	serv := NewServer(maxClients, addr)
	if err := serv.Listen(); err != nil {
		log.Fatalf("error listening: %s\n", err)
	}
	for {
		serv.Update(serverTime)
		// do simulation/process payload packets

		// send payloads to clients
		serv.Send(packetData)

		time.Sleep(deltaTime)
		serverTime += deltaTime.Seconds()
		count += 1

	}
}

func clientLoop(wg *sync.WaitGroup, packetData []byte, index int) {
	clientTime := float64(0)
	delta := float64(1.0 / 60.0)
	deltaTime := time.Duration(delta * float64(time.Second))

	// randomize start up
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

	// fake game loop
	c := NewClient(addr)
	if err := c.Connect(); err != nil {
		log.Printf("error connecting: %s\n", err)
		wg.Done()
		return
	}

	iter := 0
	for {
		c.Update(clientTime)
		// run for about 5 seconds
		if clientTime > runTime {
			log.Printf("client (%d) %d payloads, %d pings for %d iterations", index, c.PayloadCount(), c.PingCount(), iter)
			wg.Done()
			return
		}

		c.Send(packetData)
		time.Sleep(deltaTime)
		clientTime += deltaTime.Seconds()
		iter++
	}
}
