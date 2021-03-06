package main

import (
	"bytes"
	"log"
	"net"
)

// used for holding references to our clients
type clientInstance struct {
	address      *net.UDPAddr // the address of the client
	lastRecvTime float64      // last time we recv'd a packet
	payloadCount int          // how many payload we saw from this client
	pingCount    int          // how many pings we saw from this client
}

type Server struct {
	conn         *NetcodeConn      // the underlying connection
	addr         *net.UDPAddr      // the server address
	packetCh     chan *netcodeData // channel used for synchronizing packet data
	maxClients   int               // maximum number of clients
	clients      []*clientInstance // our slice of clients, pre allocated
	lastSendTime float64           // last time we sent data
	serverTime   float64           // the last serverTime/time Update was called
	sendCount    int
	printed      bool
}

// super basic ping packet
var ping = []byte("ping")

// Creates a new server, pre-allocating clients
func NewServer(maxClients int, addr *net.UDPAddr) *Server {
	s := &Server{addr: addr, maxClients: maxClients}
	s.packetCh = make(chan *netcodeData, (maxClients*MAX_PACKETS)*2)
	s.clients = make([]*clientInstance, maxClients)
	for i := 0; i < len(s.clients); i += 1 {
		s.clients[i] = &clientInstance{}
	}
	return s
}

// listens on the address that was provided to NewServer
func (s *Server) Listen() error {
	s.conn = NewNetcodeConn()
	s.conn.SetRecvHandler(s.onPacket)
	s.conn.SetMaxPackets((s.maxClients * MAX_PACKETS) * 2)
	return s.conn.Listen(s.addr)
}

// Called every 'tick' of the ficticious game loop.
// recv's data first, checks if we need to send pings, then checks if clients
// have timed out.
func (s *Server) Update(serverTime float64) {
	s.serverTime = serverTime

	// empty recv'd data from channel so we can have safe access to client instance data structures
	for {
		select {
		case recv := <-s.packetCh:
			s.OnPacketData(recv.data, recv.from)
		default:
			goto DONE
		}
	}
DONE:
	if s.lastSendTime+float64(1.0/10.0) < serverTime {
		s.Send(ping)
		s.lastSendTime = serverTime
	}

	s.checkTimeouts(serverTime)

}

// iterate over client list and check if we should remove their entry (by setting the properties to 0 values/nil)
func (s *Server) checkTimeouts(serverTime float64) {

	for i := 0; i < len(s.clients); i += 1 {
		instance := s.clients[i]
		// timeout if lastRecvTime + runTime + 1 second is < server time.
		if instance.address != nil && instance.lastRecvTime+runTime+1.0 < serverTime {
			log.Printf("client: (%s) sent %d/%d payload/pings\n", instance.address.String(), instance.payloadCount, instance.pingCount)
			instance.address = nil
			instance.lastRecvTime = 0
			instance.payloadCount = 0
			instance.pingCount = 0
			log.Printf("STATS: sent %d packets in %f\n", s.sendCount, serverTime)
		}
	}
}

// netcode conn sent us data, buffer it in our packetCh
func (s *Server) onPacket(data *netcodeData) {
	s.packetCh <- data
}

// send some payload/ping data
func (s *Server) Send(data []byte) {
	for i := 0; i < len(s.clients); i += 1 {
		if s.clients[i].address != nil {
			go s.conn.WriteTo(data, s.clients[i].address)
			s.sendCount++
		}
	}
}

// process the packet data, finds first empty entry in our client list
func (s *Server) OnPacketData(data []byte, addr *net.UDPAddr) {
	// already exists? update
	for i := 0; i < len(s.clients); i += 1 {
		instance := s.clients[i]
		if instance.address != nil && addressEqual(instance.address, addr) {
			instance.lastRecvTime = s.serverTime
			if bytes.Equal(ping, data) {
				instance.pingCount++
			} else {
				instance.payloadCount++
			}
			return
		}
	}

	// doesn't exist add a new entry unless all entries are full
	full := true
	for i := 0; i < len(s.clients); i += 1 {
		instance := s.clients[i]
		if instance.address == nil {
			full = false
			instance.address = addr
			instance.lastRecvTime = s.serverTime
			if bytes.Equal(ping, data) {
				instance.pingCount++
			} else {
				instance.payloadCount++
			}
			break
		}
	}

	if full {
		log.Printf("ignored, server full")
		return
	}
}

func addressEqual(addr1, addr2 *net.UDPAddr) bool {
	if addr1 == nil || addr2 == nil {
		return false
	}
	return addr1.IP.Equal(addr2.IP) && addr1.Port == addr2.Port
}
