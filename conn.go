package main

import (
	"errors"
	"log"
	"net"
	"sync"
)

// our struct for passing data and client addresses around
type netcodeData struct {
	data []byte
	from *net.UDPAddr
}

const (
	MAX_PACKET_BYTES   = 1220
	SOCKET_RCVBUF_SIZE = 1024 * 1024
	SOCKET_SNDBUF_SIZE = 1024 * 1024
)

// allows for supporting custom handlers
type NetcodeRecvHandler func(data *netcodeData)

type NetcodeConn struct {
	conn     *net.UDPConn  // the underlying connection
	closeCh  chan struct{} // used for closing the connection/signaling
	isClosed bool          // is this connection open/closed?

	recvSize   int // how many bytes to read
	sendSize   int // how many bytes to send
	maxBytes   int // maximum allowed bytes
	maxPackets int // maximum number of packets (not used in this version)
	xmitBuf    sync.Pool

	recvHandlerFn NetcodeRecvHandler
}

// Creates a new netcode connection
func NewNetcodeConn() *NetcodeConn {
	c := &NetcodeConn{}

	c.closeCh = make(chan struct{})
	c.isClosed = true

	c.maxBytes = MAX_PACKET_BYTES
	c.maxPackets = MAX_PACKETS
	c.recvSize = SOCKET_RCVBUF_SIZE
	c.sendSize = SOCKET_SNDBUF_SIZE

	return c
}

func (c *NetcodeConn) SetMaxPackets(max int) {
	if !c.isClosed {
		log.Printf("unable to set max packets after connect/listen called")
		return
	}
	c.maxPackets = max
}

func (c *NetcodeConn) SetRecvHandler(recvHandlerFn NetcodeRecvHandler) {
	c.recvHandlerFn = recvHandlerFn
}

func (c *NetcodeConn) Write(b []byte) (int, error) {
	if c.isClosed {
		return -1, errors.New("unable to write, socket has been closed")
	}
	return c.conn.Write(b)
}

func (c *NetcodeConn) WriteTo(b []byte, to *net.UDPAddr) (int, error) {
	if c.isClosed {
		return -1, errors.New("unable to write, socket has been closed")
	}
	return c.conn.WriteToUDP(b, to)
}

func (c *NetcodeConn) Close() error {
	if !c.isClosed {
		close(c.closeCh)
	}
	c.isClosed = true
	return c.conn.Close()
}

func (c *NetcodeConn) SetReadBuffer(bytes int) {
	c.recvSize = bytes
}

func (c *NetcodeConn) SetWriteBuffer(bytes int) {
	c.sendSize = bytes
}

// LocalAddr returns the local network address.
func (c *NetcodeConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *NetcodeConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *NetcodeConn) Dial(address *net.UDPAddr) error {
	var err error

	if c.recvHandlerFn == nil {
		return errors.New("packet handler must be set before calling listen")
	}

	c.closeCh = make(chan struct{})
	c.conn, err = net.DialUDP(address.Network(), nil, address)
	if err != nil {
		return err
	}
	return c.create()
}

func (c *NetcodeConn) Listen(address *net.UDPAddr) error {
	var err error

	if c.recvHandlerFn == nil {
		return errors.New("packet handler must be set before calling listen")
	}

	c.conn, err = net.ListenUDP(address.Network(), address)
	if err != nil {
		return err
	}

	c.create()
	return err
}

func (c *NetcodeConn) create() error {
	c.isClosed = false
	c.xmitBuf.New = func() interface{} {
		return make([]byte, c.maxBytes)
	}
	c.conn.SetReadBuffer(c.recvSize)
	c.conn.SetWriteBuffer(c.sendSize)
	go c.readLoop()
	return nil
}

func (c *NetcodeConn) receiver(ch chan *netcodeData) {
	for {
		if netData, err := c.read(); err == nil {
			select {
			case ch <- netData:
			case <-c.closeCh:
				return
			}
		} else {
			log.Printf("error reading data from socket: %s\n", err)
		}

	}
}

// read does the actual connection read call, verifies we have a
// buffer > 0 and < maxBytes before we bother to attempt to actually
// dispatch it to the recvHandlerFn.
func (c *NetcodeConn) read() (*netcodeData, error) {
	var n int
	var from *net.UDPAddr
	var err error

	data := c.xmitBuf.Get().([]byte)[:c.maxBytes]
	n, from, err = c.conn.ReadFromUDP(data)
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, errors.New("socket error: 0 byte length recv'd")
	}

	if n > c.maxBytes {
		return nil, errors.New("packet size was > maxBytes")
	}
	netData := &netcodeData{}
	netData.data = data[:n]
	netData.from = from
	return netData, nil
}

// dispatch the netcodeData to the bound recvHandler function.
func (c *NetcodeConn) readLoop() {
	dataCh := make(chan *netcodeData, c.maxPackets)
	go c.receiver(dataCh)

	for {
		select {
		case data := <-dataCh:
			//c.conn.WriteTo(data.data, data.from)
			c.recvHandlerFn(data)
		case <-c.closeCh:
			return
		}
	}
}
