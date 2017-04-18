package main

import (
	"log"
	"net"
)

type clientInstance struct {
	address      *net.UDPAddr
	lastRecvTime float64
	payloadCount int
}

type Server struct {
	conn         *NetcodeConn
	addr         *net.UDPAddr
	packetCh     chan *netcodeData
	maxClients   int
	clients      []*clientInstance
	lastSendTime float64
	serverTime   float64
}

var ping = []byte("ping")

func NewServer(maxClients int, addr *net.UDPAddr) *Server {
	s := &Server{addr: addr, maxClients: maxClients}
	s.packetCh = make(chan *netcodeData, (maxClients*MAX_PACKETS)*2)
	s.clients = make([]*clientInstance, maxClients)
	for i := 0; i < len(s.clients); i += 1 {
		s.clients[i] = &clientInstance{}
	}
	return s
}

func (s *Server) Listen() error {
	s.conn = NewNetcodeConn()
	s.conn.SetRecvHandler(s.onPacket)
	s.conn.SetMaxPackets((s.maxClients * MAX_PACKETS) * 2)
	return s.conn.Listen(s.addr)
}

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

func (s *Server) checkTimeouts(serverTime float64) {
	for i := 0; i < len(s.clients); i += 1 {
		instance := s.clients[i]
		// timeout if lastRecvTime + runTime + 1 second is > server time.
		if instance.address != nil && instance.lastRecvTime+runTime+1.0 < serverTime {
			log.Printf("client: (idx: %d) sent %d payload & pings\n", i, instance.payloadCount)
			instance.address = nil
			instance.lastRecvTime = 0
			instance.payloadCount = 0
		}
	}
}

func (s *Server) onPacket(data *netcodeData) {
	s.packetCh <- data
}

func (s *Server) Send(data []byte) {
	for i := 0; i < len(s.clients); i += 1 {
		if s.clients[i].address != nil {
			go s.conn.WriteTo(data, s.clients[i].address)
		}
	}
}

func (s *Server) OnPacketData(data []byte, addr *net.UDPAddr) {
	full := true

	// basic client checks
	for i := 0; i < len(s.clients); i += 1 {
		instance := s.clients[i]
		if instance.address == nil {
			full = false
			instance.address = addr
			instance.lastRecvTime = s.serverTime
			instance.payloadCount++
			break
		} else if addressEqual(instance.address, addr) {
			full = false
			instance.lastRecvTime = s.serverTime
			instance.payloadCount++
			break
		}
	}

	if full {
		log.Printf("ignored, server full")
		return
	}

	//log.Printf("got data: len(%d) from %s\n", len(data), addr.String())
}

func addressEqual(addr1, addr2 *net.UDPAddr) bool {
	if addr1 == nil || addr2 == nil {
		return false
	}
	return addr1.IP.Equal(addr2.IP) && addr1.Port == addr2.Port
}
