package api

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vite/net"
	"strconv"
)

type NetApi struct {
	p2p *p2p.Server
	net net.Net
	log log15.Logger
}

func NewNetApi(vite *vite.Vite) *NetApi {
	return &NetApi{
		p2p: vite.P2P(),
		net: vite.Net(),
		log: log15.New("module", "rpc_api/net_api"),
	}
}

type SyncInfo struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Received string `json:"received"`
	Current  string `json:"current"`
	State    uint   `json:"state"`
	Status   string `json:"status"`
}

func (n *NetApi) SyncInfo() *SyncInfo {
	log.Info("SyncInfo")
	s := n.net.Status()

	return &SyncInfo{
		From:     strconv.FormatUint(s.From, 10),
		To:       strconv.FormatUint(s.To, 10),
		Received: strconv.FormatUint(s.Received, 10),
		Current:  strconv.FormatUint(s.Current, 10),
		State:    uint(s.State),
		Status:   s.State.String(),
	}
}

func (n *NetApi) Peers() *net.NodeInfo {
	return n.net.Info()
}

func (n *NetApi) PeersCount() uint {
	info := n.net.Info()
	return uint(len(info.Peers))
}

func (n *NetApi) Connect(ip_port string) error {
	if n.p2p == nil {
		return errors.New("can not get p2p server")
	}
	return n.p2p.Connect(ip_port)
}
