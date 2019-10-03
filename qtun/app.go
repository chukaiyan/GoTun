package qtun

import (
	"log"
	"net"
	"sync"
	"github.com/chukaiyan/GoTun/globalconfig"
	"github.com/chukaiyan/GoTun/config"
	"github.com/chukaiyan/GoTun/iface"
	"github.com/chukaiyan/GoTun/protocol"
	"github.com/chukaiyan/GoTun/transport"
	"github.com/golang/protobuf/proto"
)

type App struct {
	//config config.Config
	client *Peer
	routes map[string]Route
	mutex  sync.RWMutex
	server *transport.Server
	iface  *iface.Iface
}

func NewApp(config config.Config) *App {

	var anewapp App
	globalconfig.Globalconfigset = config;
	anewapp = App{
		//config: config,
		routes: make(map[string]Route),
	}
	if globalconfig.Globalconfigset.Verbose == 3 {
		log.Printf("chuxu L3 NewApp  finished ")


	}
	return &anewapp; 
}

func (a *App) Run() error {
	if globalconfig.Globalconfigset.ServerMode == 1 {
		a.server = transport.NewServer(globalconfig.Globalconfigset.Listen, "not_use_private_address", a, globalconfig.Globalconfigset.Key)
		a.server.SetConfig(globalconfig.Globalconfigset)
		go a.server.Start()
	} else {
		err := a.InitClient()
		if err != nil {
			return err
		}
	}

	return a.StartFetchTunInterface()
}

func (a *App) StartFetchTunInterface() error {
	a.iface = iface.New("", globalconfig.Globalconfigset.Ip, globalconfig.Globalconfigset.Mtu)
	err := a.iface.Start()
	if err != nil {
		return err
	}

	for i := 0; i < 10; i++ {
		go a.FetchAndProcessTunPkt()
	}

	return a.FetchAndProcessTunPkt()
}

func (a *App) FetchAndProcessTunPkt() error {
	pkt := iface.NewPacketIP(globalconfig.Globalconfigset.Mtu)
	for {
		n, err := a.iface.Read(pkt)
		if err != nil {
			log.Printf("FetchAndProcessTunPkt::read ip pkt error: %v", err)
			return err
		}
		src := pkt.GetSourceIP().String()
		dst := pkt.GetDestinationIP().String()
		if globalconfig.Globalconfigset.Verbose == 1 {
			log.Printf("FetchAndProcessTunPkt::got tun packet: src=%s dst=%s len=%d", src, dst, n)
		}
		if globalconfig.Globalconfigset.ServerMode == 1 {
			// log.Printf("FetchAndProcessTunPkt::receiver tun packet dst address  dst=%s, route_local_addr=%s", dst, a.routes[dst].LocalAddr)
			conn := a.server.GetConnsByAddr(a.routes[dst].LocalAddr)
			if conn == nil {
				if globalconfig.Globalconfigset.Verbose == 1 {
					log.Printf("FetchAndProcessTunPkt::unknown destination, packet dropped src=%s,dst=%s", src, dst)
				}
			} else {
				conn.SendPacket(pkt)
			}
		} else {
			//client send packet
			a.client.SendPacket(pkt)
		}
	}
}

func (a *App) InitClient() error {
	//For server no need to make connection to client -gs
	// if a.config.ServerMode == 0 {
	peer := NewPeer(globalconfig.Globalconfigset, a)
	peer.Start()
	a.client = peer
	return nil
}

func (a *App) getRoutes() []Route {
	a.mutex.Lock()
	routes := make([]Route, len(a.routes))
	i := 0
	for _, route := range a.routes {
		routes[i] = route
		i++
	}
	a.mutex.Unlock()
	return routes
}

func (a *App) OnData(buf []byte, conn *net.TCPConn) {
	ep := protocol.Envelope{}
	err := proto.Unmarshal(buf, &ep)
	if err != nil {
		log.Printf("OnData::proto unmarshal err: %s", err)
		return
	}
	switch ep.Type.(type) {
	case *protocol.Envelope_Ping:
		ping := ep.GetPing()
		//log.Printf("received ping: %s", ping.String())
		//根据Client发来的Ping包信息来添加路由
		a.mutex.Lock()
		a.routes[ping.GetIP()] = Route{
			LocalAddr: ping.GetLocalAddr(),
			IP:        ping.GetIP(),
		}

		a.server.SetConns(a.routes[ping.GetIP()].LocalAddr, conn)
		if globalconfig.Globalconfigset.Verbose == 1 {
			log.Printf("OnData::routes %s", a.routes)
		}
		a.mutex.Unlock()
	case *protocol.Envelope_Packet:
		pkt := iface.PacketIP(ep.GetPacket().GetPayload())
		if globalconfig.Globalconfigset.Verbose == 3 {  //chuxu changed to 3
			log.Printf("OnData::received packet: src=%s dst=%s len=%d",
				pkt.GetSourceIP(), pkt.GetDestinationIP(), len(pkt))
		}
		a.iface.Write(pkt)
	}
}
