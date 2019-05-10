package proto

import (
	"encoding/json"
	"os"
	"path"
	"strings"
)

/*

command name must consist of 4 bytes
all commands declarated in proto/commands.go
commands:

INFO                   - request user info
LIST                   - request for peer list
SEND msg               - send message with text msg
FILE name data         - send part of file
SEEK DHState           - request for ip of peer with DHState
DISC                   - notification about disconnection

REFO data              - response for INFO request
RELI data              - response for LIST request
RESE data              - response for SEEK request

*/

type PeerHandler func(*Host, *Peer, []byte) error

var PeerCommands = peerCommands()

func peerCommands() *CommandParser {
	parser := CreateCommandParser(func(i interface{}) bool { _, ok := i.(PeerHandler); return ok })
	parser.AddCommand(info, PeerHandler(peerInfoHandler))
	parser.AddCommand(list, PeerHandler(peerListHandler))
	parser.AddCommand(send, PeerHandler(peerSendHandler))
	parser.AddCommand(file, PeerHandler(peerFileHandler))
	parser.AddCommand(disc, PeerHandler(peerDiscHandler))
	parser.AddCommand(refo, PeerHandler(peerRefoHandler))
	parser.AddCommand(reli, PeerHandler(peerReliHandler))
	return parser
}

func peerInfoHandler(host *Host, peer *Peer, data []byte) error {
	js, _ := json.Marshal(host)
	return sendCommand(peer, refo, js)
}

func peerListHandler(host *Host, peer *Peer, data []byte) error {
	js, _ := json.Marshal(host.Peers)
	return sendCommand(peer, reli, js)
}

func peerSendHandler(host *Host, peer *Peer, data []byte) error {
	host.Msg <- peer.Login + ": " + string(data)
	return nil
}

func peerFileHandler(host *Host, peer *Peer, data []byte) error {
	fstruct := &File{}
	err := json.Unmarshal(data, fstruct)
	if err != nil {
		return err
	}

	// Create directory with name of peer
	file := path.Join("./", peer.Login)
	err = os.MkdirAll(file, 0777)
	if err != nil {
		return err
	}

	// Create file ./Login/file
	file = path.Join(file, fstruct.Name)
	fd, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()

	host.Msg <- peer.Login + ": part of " + fstruct.Name + " was received"
	_, err = fd.Write(fstruct.Data)
	return err
}

func peerDiscHandler(host *Host, peer *Peer, data []byte) error {
	peer.Close()
	peer.Crypto = nil
	return nil
}

func peerRefoHandler(host *Host, peer *Peer, data []byte) error {
	p := &Peer{}
	err := json.Unmarshal(data, p)
	if err == nil {
		ip := strings.Split(peer.Conn.RemoteAddr().String(), ":")[0]
		host.Peers[ip].Login = p.Login
		host.Peers[ip].Addr = p.Addr
	}
	return err
}

func peerReliHandler(host *Host, peer *Peer, data []byte) error {
	peers := make(map[string]*Peer)
	err := json.Unmarshal(data, peers)
	if err != nil {
		return err
	}
	for k, v := range peers {
		_, ok := host.Peers[k]
		if !ok {
			host.Peers[k] = v
		}
	}
	return nil
}
