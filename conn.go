package main

import (
	"errors"
	"log"
	"net"
)

type netcodeData struct {
	data []byte
	from *net.UDPAddr
}

const (
	MAX_PACKET_BYTES   = 1220
	SOCKET_RCVBUF_SIZE = 1024 * 1024
	SOCKET_SNDBUF_SIZE = 1024 * 1024
)

type NetcodeRecvHandler func(data *netcodeData)

type NetcodeConn struct {
	conn     *net.UDPConn
	closeCh  chan struct{}
	isClosed bool

	recvSize   int
	sendSize   int
	maxBytes   int
	maxPackets int

	recvHandlerFn NetcodeRecvHandler
}

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
	return c.conn.WriteTo(b, to)
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
// buffer > 0 and < maxBytes and is of a valid packet type before
// we bother to attempt to actually dispatch it to the recvHandlerFn.
func (c *NetcodeConn) read() (*netcodeData, error) {
	var n int
	var from *net.UDPAddr
	var err error
	netData := &netcodeData{}
	netData.data = make([]byte, c.maxBytes)

	n, from, err = c.conn.ReadFromUDP(netData.data)
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, errors.New("socket error: 0 byte length recv'd")
	}

	if n > c.maxBytes {
		return nil, errors.New("packet size was > maxBytes")
	}

	netData.data = netData.data[:n]
	netData.from = from
	return netData, nil
}

// dispatch the netcodeData to the bound recvHandler function.
func (c *NetcodeConn) readLoop() {
	dataCh := make(chan *netcodeData)
	go c.receiver(dataCh)
	for {
		select {
		case data := <-dataCh:
			c.recvHandlerFn(data)
		case <-c.closeCh:
			return
		}
	}
}
