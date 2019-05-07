package proto

import (
	"encoding/json"
	"errors"
)

/*

command name must consist of 4 bytes
all commands declarated in proto/commands.go
commands:

CONN ip    - initiate connection with peer
LIST       - request for peer list
SEND msg   - send message to peer with appropriate ip address
FILE path  - send file to peer with appropriate ip address
SEEK login - request for peers with appropriate login

RELI data  - response for LIST request
RESE data  - response for SEEK request
REER data  - response with error

*/

type Manager struct {
	Conn     *Conn
	Peer     *Peer
	Commands *CommandParser
}

type ManagerHandler func(*Host, *Manager, []byte) error

var ManagerCommands = managerCommands()

func managerCommands() *CommandParser {
	parser := CreateCommandParser(func(i interface{}) bool { _, ok := i.(ManagerHandler); return ok })
	parser.AddCommand(conn, ManagerHandler(managerConnHandler))
	parser.AddCommand(list, ManagerHandler(managerListHandler))
	parser.AddCommand(send, ManagerHandler(managerSendHandler))
	return parser
}

func (manager *Manager) SendMessage(data []byte) error {
	_, err := manager.Conn.WritePackage(packCommand(send, data))
	return err
}

// Manager commands
func managerConnHandler(host *Host, manager *Manager, data []byte) error {
	conn, err := Dial("tcp", string(data))
	if err != nil {
		return err
	}

	manager.Peer, err = host.DialPeer(conn)
	if err != nil {
		return err
	}

	go func() {
		host.Commands.CommandLoop(manager.Peer, func(handler interface{}, arg []byte) error {
			h := handler.(PeerHandler)
			return h(host, manager.Peer, arg)
		})
	}()
	manager.Peer.WritePackage(packCommand(info, nil))
	return nil
}

func managerListHandler(host *Host, manager *Manager, data []byte) error {
	js, _ := json.Marshal(host.Peers)
	_, err := manager.Conn.WritePackage(packCommand(reli, js))
	return err
}

func managerSendHandler(host *Host, manager *Manager, data []byte) error {
	if manager.Peer == nil {
		return errors.New("can't send to unknown peer")
	}

	_, err := manager.Peer.WritePackage(packCommand(send, data))
	return err
}
