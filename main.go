package main

import (
	"flag"
	//"github.com/pkg/profile"
	"log"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"time"
)

const MAX_PACKETS = 256

var serverMode bool
var maxClients int
var runTime float64
var runProfiler bool
var runTrace bool
var runMem bool
var runCpu bool

var addr = &net.UDPAddr{IP: net.ParseIP("::1"), Port: 40000}

func init() {
	flag.BoolVar(&serverMode, "server", false, "pass this flag to run the server")
	flag.BoolVar(&runProfiler, "prof", false, "pass this flag to enable profiling")
	flag.IntVar(&maxClients, "num", 64, "number of clients to serve or to create")
	flag.Float64Var(&runTime, "runtime", 5.0, "how long to run clients for/clear client buffer in seconds")

}

func main() {
	flag.Parse()
	packetData := make([]byte, 1220)
	for i := 0; i < len(packetData); i += 1 {
		packetData[i] = byte(i)
	}

	if runProfiler {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
		//p := profile.Start(profile.TraceProfile, profile.ProfilePath("."), profile.NoShutdownHook)
		//defer p.Stop()
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
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	serverTime := float64(0)
	delta := float64(1.0 / 60.0)
	deltaTime := time.Duration(delta * float64(time.Second))
	count := 0

	serv := NewServer(maxClients, addr)
	if err := serv.Listen(); err != nil {
		log.Fatalf("error listening: %s\n", err)
	}
	for {
		select {
		case <-c:
			return
		default:
		}

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
			log.Printf("client (%d) %drx/%dtx, %d pings\n", index, c.PayloadCount(), iter, c.PingCount())
			wg.Done()
			return
		}

		c.Send(packetData)
		time.Sleep(deltaTime)
		clientTime += deltaTime.Seconds()
		iter++
	}
}
