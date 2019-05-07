package proto

import (
	"encoding/binary"
	"errors"
	"net"
	"strings"
)

type Host struct {
	Login    string
	DifHel   *DHState         `json:"-"`
	Peers    map[string]*Peer `json:"-"`
	Commands *CommandParser   `json:"-"`
	Msg      chan string      `json:"-"`
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

func Dial(network, address string) (*Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return &Conn{conn: conn}, nil
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
	//peer.Crypto.ExchangeKeys()

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

	return peer, nil
}