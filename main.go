package main

import (
	"./proto"
	"log"
	"strconv"
	"strings"
)

func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("config is loaded")

	host := &proto.Host{
		Login:    config.Login,
		DifHel:   config.DifHel,
		Peers:    config.Peers,
		Commands: proto.PeerCommands,
		Msg:      make(chan string),
	}

	ln, err := proto.Listen("tcp", ":"+strconv.Itoa(int(config.Port)))
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("server is started")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("%s is connected\n", conn.RemoteAddr().String())

		go Distribute(host, conn)
	}
}

func Distribute(host *proto.Host, conn *proto.Conn) {
	// If local connection, it's manager, else it's another peer
	if strings.HasPrefix(conn.RemoteAddr().String(), "127.0.0.1") {
		ManagerHandler(host, conn)
	} else {
		PeerHandler(host, conn)
	}
}

func ManagerHandler(host *proto.Host, conn *proto.Conn) {
	manager := &proto.Manager{
		Conn:     conn,
		Commands: proto.ManagerCommands,
	}
	go SendToManager(host, manager)

	manager.Commands.CommandLoop(manager.Conn, func(handler interface{}, arg []byte) error {
		h := handler.(proto.ManagerHandler)
		return h(host, manager, arg)
	})
}

func SendToManager(host *proto.Host, manager *proto.Manager) {
	for {
		msg := <-host.Msg
		err := manager.SendMessage([]byte(msg))
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func PeerHandler(host *proto.Host, conn *proto.Conn) {
	peer, err := host.AcceptPeer(conn)
	if err != nil {
		log.Println(err)
		return
	}

	host.Commands.CommandLoop(peer, func(handler interface{}, arg []byte) error {
		h := handler.(proto.PeerHandler)
		return h(host, peer, arg)
	})
}
