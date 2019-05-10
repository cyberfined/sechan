package main

import (
	"./proto"
	"encoding/json"
	"log"
	"net"
	"strings"
	"time"
)

func SendInfo(host *proto.Host, addr string) {
	conn, err := proto.Dial("udp", addr)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	js, _ := json.Marshal(host)
	for {
		_, err = conn.WritePackage(js)
		if err != nil {
			log.Println(err)
		}
		time.Sleep(5 * time.Second)
	}
}

func ReceiveInfo(host *proto.Host, addr string) {
	udpaddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalln(err)
	}

	udpconn, err := net.ListenMulticastUDP("udp", nil, udpaddr)
	if err != nil {
		log.Fatalln(err)
	}
	conn := proto.CreateConn(udpconn)
	defer conn.Close()

	for {
		buf, err := conn.ReadPackage()
		if err != nil {
			log.Println(err)
			continue
		}

		peer := &proto.Peer{}
		err = json.Unmarshal(buf, peer)
		if err != nil {
			log.Println(err)
			continue
		}

		if peer.Addr == host.Addr {
			continue
		}

		ip := strings.Split(peer.Addr, ":")[0]
		_, ok := host.Peers[ip]
		if !ok {
			host.Peers[ip] = peer
			SavePeers(host.Peers)
		}
	}
}
