package proto

import (
	"encoding/binary"
	"errors"
	"net"
	"strings"
)

const MaxPacketSize uint32 = 65536

var ErrLongPacket = errors.New("packet is too long")

type Host struct {
	Login    string
	Addr     string
	DifHel   *DHState         `json:"-"`
	Peers    map[string]*Peer `json:"-"`
	Commands *CommandParser   `json:"-"`
	Msg      chan string      `json:"-"`
	Quit     chan bool        `json:"-"`
}

type PackageReadWriter interface {
	ReadPackage() ([]byte, error)
	WritePackage([]byte) (int, error)
}

type Conn struct {
	conn net.Conn
}

type Listener struct {
	listener net.Listener
}

func CreateConn(conn net.Conn) *Conn {
	return &Conn{
		conn: conn,
	}
}

func Dial(network, address string) (*Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return &Conn{conn: conn}, nil
}

func CreateListener(listener net.Listener) *Listener {
	return &Listener{
		listener: listener,
	}
}

func Listen(network, address string) (*Listener, error) {
	ln, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return &Listener{listener: ln}, nil
}

func (c *Conn) WritePackage(data []byte) (int, error) {
	lbuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lbuf, uint32(len(data)))

	_, err := c.conn.Write(lbuf)
	if err != nil {
		return 0, err
	}

	return c.conn.Write(data)
}

func (c *Conn) ReadPackage() ([]byte, error) {
	lbuf := make([]byte, 4)
	n, err := c.conn.Read(lbuf)
	if err != nil {
		return nil, err
	}
	if n != 4 {
		return nil, errors.New("wrong length")
	}

	length := binary.LittleEndian.Uint32(lbuf)
	if length > MaxPacketSize {
		return nil, ErrLongPacket
	}

	buf := make([]byte, length)
	n, err = c.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if uint32(n) != length {
		return nil, errors.New("wrong length")
	}

	return buf, nil

}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

func (ln *Listener) Accept() (*Conn, error) {
	conn, err := ln.listener.Accept()
	if err != nil {
		return nil, err
	}
	return &Conn{conn: conn}, nil
}

func (ln *Listener) Close() error {
	return ln.listener.Close()
}

func (ln *Listener) Addr() net.Addr {
	return ln.listener.Addr()
}

func (host *Host) DialPeer(conn *Conn) (*Peer, error) {
	addr := conn.RemoteAddr().String()
	ip := strings.Split(addr, ":")[0]

	peer := &Peer{
		Addr:   addr,
		DifHel: &DHState{},
		Conn:   conn,
	}
	v, ok := host.Peers[ip]
	if ok {
		peer.Login = v.Login
		peer.Addr = v.Addr
		peer.Crypto = v.Crypto
	}

	if peer.Crypto == nil {
		key, err := peer.DifHel.PassiveDHExchange(peer.Conn)
		if err != nil {
			return nil, err
		}

		peer.Crypto = InitCryptoState(key, true)
	}

	if !ok {
		host.Peers[ip] = peer
	}

	sendCommand(peer, info, nil)
	sendCommand(peer, list, nil)
	return peer, nil
}

func (host *Host) AcceptPeer(conn *Conn) (*Peer, error) {
	addr := conn.RemoteAddr().String()
	ip := strings.Split(addr, ":")[0]

	peer := &Peer{
		Addr:   addr,
		DifHel: &DHState{},
	}
	v, ok := host.Peers[ip]
	if ok {
		peer.Login = v.Login
		peer.Addr = v.Addr
		peer.Crypto = v.Crypto
	}
	peer.Conn = conn

	if peer.Crypto == nil {
		key, err := host.DifHel.ActiveDHExchange(peer.Conn)
		if err != nil {
			return nil, err
		}

		peer.Crypto = InitCryptoState(key, false)
	}

	if !ok {
		host.Peers[ip] = peer
	}

	sendCommand(peer, info, nil)
	sendCommand(peer, list, nil)
	return peer, nil
}

func (host *Host) Disconnect() {
	for _, p := range host.Peers {
		if p.Conn == nil {
			continue
		}
		sendCommand(p, disc, nil)
		p.Close()
		p.Crypto = nil
	}
}
