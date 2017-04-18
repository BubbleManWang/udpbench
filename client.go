package main

import (
	"bytes"
	"net"
	"sync/atomic"
)

// our client
type Client struct {
	serverAddr   *net.UDPAddr
	conn         *NetcodeConn
	packetCh     chan *netcodeData // channel for recv'ing packets from server
	lastSendTime float64
	payloadCount uint32
	pingCount    uint32
}

func NewClient(serverAddr *net.UDPAddr) *Client {
	c := &Client{}
	c.serverAddr = serverAddr
	c.packetCh = make(chan *netcodeData, MAX_PACKETS)
	return c
}

// connect to the server, returning error if dial fails.
func (c *Client) Connect() error {
	c.conn = NewNetcodeConn()
	c.conn.SetRecvHandler(c.onPacket)
	return c.conn.Dial(c.serverAddr)
}

// when a packet from the netcodeConn is recv'd, we push it to our buffered packet data
func (c *Client) onPacket(data *netcodeData) {
	c.packetCh <- data
}

// our game loop tick, recv data first, then send ping if we need to.
func (c *Client) Update(clientTime float64) {
	// empty recv'd data from channel so we can have safe access to client manager data structures
	for {
		select {
		case recv := <-c.packetCh:
			c.OnPacketData(recv.data, recv.from)
		default:
			goto DONE
		}
	}
DONE:
	if c.lastSendTime+float64(1.0/10.0) < clientTime {
		c.lastSendTime = clientTime
		c.Send(ping)
	}
}

// send data to the server
func (c *Client) Send(data []byte) {
	c.conn.Write(data)
}

// do actual processing of packet data here.
func (c *Client) OnPacketData(data []byte, from *net.UDPAddr) {
	if !addressEqual(c.serverAddr, addr) {
		return
	}

	if bytes.Equal(ping, data) {
		atomic.AddUint32(&c.pingCount, 1)
		return
	}

	atomic.AddUint32(&c.payloadCount, 1)
}

// returns # of payloads server sent to us
func (c *Client) PayloadCount() uint32 {
	return atomic.LoadUint32(&c.payloadCount)
}

// returns # of pings server sent to us
func (c *Client) PingCount() uint32 {
	return atomic.LoadUint32(&c.pingCount)
}
