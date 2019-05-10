package main

import (
	"./proto"
	"encoding/json"
	"io/ioutil"
	"time"
)

const (
	configFile = "config"
	stateFile  = "state"
	peersFile  = "peers"
)

type Config struct {
	UserConfig
	DHStateConfig
	Peers map[string]*proto.Peer
}

type UserConfig struct {
	Login string
	Addr  string
	Port  string
}

type DHStateConfig struct {
	DifHel  *proto.DHState
	Created time.Time
}

func LoadUserConfig() (*UserConfig, error) {
	buf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	uc := &UserConfig{}
	err = json.Unmarshal(buf, uc)
	if err != nil {
		return nil, err
	}

	return uc, nil
}

func LoadDHStateConfig() (*DHStateConfig, error) {
	var (
		dh         *DHStateConfig
		difference time.Duration
	)

	dh = &DHStateConfig{}

	buf, err := ioutil.ReadFile(stateFile)
	if err != nil {
		goto Gen
	}

	err = json.Unmarshal(buf, dh)
	if err != nil {
		goto Gen
	}

	difference = time.Now().Sub(dh.Created)
	if difference.Hours() >= 1860 {
		goto Gen
	}

	return dh, nil
Gen:
	dh.DifHel, err = proto.InitDHState()
	if err != nil {
		return nil, err
	}
	dh.Created = time.Now()

	buf, _ = json.Marshal(dh)
	ioutil.WriteFile(stateFile, buf, 0644)
	return dh, nil
}

func LoadPeers() map[string]*proto.Peer {
	buf, err := ioutil.ReadFile(peersFile)
	if err != nil {
		return make(map[string]*proto.Peer)
	}

	var peers map[string]*proto.Peer
	err = json.Unmarshal(buf, peers)
	if err != nil {
		return make(map[string]*proto.Peer)
	}

	return peers
}

func SavePeers(peers map[string]*proto.Peer) {
	buf, _ := json.Marshal(peers)
	ioutil.WriteFile(peersFile, buf, 0644)
}

func LoadConfig() (*Config, error) {
	uc, err := LoadUserConfig()
	if err != nil {
		return nil, err
	}

	dh, err := LoadDHStateConfig()
	if err != nil {
		return nil, err
	}

	peers := LoadPeers()

	return &Config{
		UserConfig:    *uc,
		DHStateConfig: *dh,
		Peers:         peers,
	}, nil
}
