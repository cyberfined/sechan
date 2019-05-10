package proto

import (
	"encoding/json"
	"errors"
	"os"
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
QUIT       - quit

RELI data  - response for LIST request
RESE data  - response for SEEK request
REER data  - response with error

*/

const fileBufReserved = 1024

type Manager struct {
	Conn     *Conn
	Peer     *Peer
	Commands *CommandParser
}

type File struct {
	Name string
	Data []byte
}

type ManagerHandler func(*Host, *Manager, []byte) error

var ManagerCommands = managerCommands()

func managerCommands() *CommandParser {
	parser := CreateCommandParser(func(i interface{}) bool { _, ok := i.(ManagerHandler); return ok })
	parser.AddCommand(conn, ManagerHandler(managerConnHandler))
	parser.AddCommand(disc, ManagerHandler(managerDiscHandler))
	parser.AddCommand(list, ManagerHandler(managerListHandler))
	parser.AddCommand(send, ManagerHandler(managerSendHandler))
	parser.AddCommand(file, ManagerHandler(managerFileHandler))
	parser.AddCommand(quit, ManagerHandler(managerQuitHandler))
	return parser
}

func (manager *Manager) SendMessage(msg []byte) error {
	return sendCommand(manager.Conn, send, msg)
}

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

	sendCommand(manager.Peer, info, nil)
	return nil
}

func managerDiscHandler(host *Host, manager *Manager, data []byte) error {
	if manager.Peer != nil {
		manager.Peer.Close()
		manager.Peer = nil
	}
	return nil
}

func managerListHandler(host *Host, manager *Manager, data []byte) error {
	js, _ := json.Marshal(host.Peers)
	return sendCommand(manager.Conn, reli, js)
}

func managerSendHandler(host *Host, manager *Manager, data []byte) error {
	if manager.Peer == nil {
		return errors.New("can't send to unknown peer")
	}

	return sendCommand(manager.Peer, send, data)
}

func managerFileHandler(host *Host, manager *Manager, data []byte) error {
	if manager.Peer == nil {
		return errors.New("can't send to unknown peer")
	}

	fd, err := os.Open(string(data))
	if err != nil {
		return err
	}
	defer fd.Close()

	stat, err := fd.Stat()
	if err != nil {
		return err
	}

	chunkSize := int64(MaxPacketSize - CommandLength - fileBufReserved)
	chunks := stat.Size() / chunkSize
	if stat.Size()%chunkSize != 0 {
		chunks++
	}

	fstruct := File{
		Name: string(data),
	}
	buf := make([]byte, chunkSize+fileBufReserved)
	for ; chunks > 0; chunks-- {
		n, err := fd.Read(buf)
		if err != nil {
			return err
		}
		fstruct.Data = buf[:n]
		js, _ := json.Marshal(fstruct)
		err = sendCommand(manager.Peer, file, js)
		if err != nil {
			return err
		}
	}
	return nil
}

func managerQuitHandler(host *Host, manager *Manager, data []byte) error {
	host.Quit <- true
	return nil
}
