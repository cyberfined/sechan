package proto

import "encoding/json"

/*

command name must consist of 4 bytes
all commands declarated in proto/commands.go
commands:

INFO                   - request user info
LIST                   - request for peer list
SEND msg               - send message with text msg
FILE name max cur data - send cur of max part of file
SEEK DHState           - request for ip of peer with DHState

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
	parser.AddCommand(refo, PeerHandler(peerRefoHandler))
	parser.AddCommand(reli, PeerHandler(peerReliHandler))
	return parser
}

func peerInfoHandler(host *Host, peer *Peer, data []byte) error {
	js, _ := json.Marshal(host)
	_, err := peer.WritePackage(packCommand(refo, js))
	return err
}

func peerListHandler(host *Host, peer *Peer, data []byte) error {
	js, _ := json.Marshal(host.Peers)
	_, err := peer.WritePackage(packCommand(reli, js))
	return err
}

func peerSendHandler(host *Host, peer *Peer, data []byte) error {
	host.Msg <- peer.Login + ": " + string(data)
	return nil
}

func peerRefoHandler(host *Host, peer *Peer, data []byte) error {
	err := json.Unmarshal(data, peer)
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
