package iface

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/songgao/water"
)

type Iface struct {
	name string
	ip   string
	mtu  int
	ifce *water.Interface
}

func New(name, ip string, mtu int) *Iface {
	return &Iface{
		name: name,
		ip:   ip,
		mtu:  mtu,
	}
}

func (i *Iface) Start() error {
	ip, netIP, err := net.ParseCIDR(i.ip)
	if err != nil {
		return err
	}
	config := water.Config{
		DeviceType: water.TUN,
	}

	i.ifce, err = water.New(config)
	if err != nil {
		return err
	}

	log.Printf("tun interface name: %s", i.ifce.Name())
	mask := netIP.Mask
	netmask := fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("ifconfig", i.Name(),
			ip.String(), ip.String(), "netmask", netmask,
			"mtu", strconv.Itoa(i.mtu), "up")
	} else {
		cmd = exec.Command("ifconfig", i.Name(),
			ip.String(), "netmask", netmask,
			"mtu", strconv.Itoa(i.mtu), "up")
	}

	log.Printf("chu T9 ifconfig %s %s %s %s %s %s %s %s", i.Name(), ip.String(), ip.String(), "netmask", netmask, "mtu", strconv.Itoa(i.mtu), "up")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("run ifconfig fail: %v, %s", err, string(output))
		return fmt.Errorf("err: %s %s", err, string(output))
	}

	if runtime.GOOS == "darwin" {
		i.AddSysRoute(&ip)
	}

	return nil
}

func (i *Iface) AddSysRoute(ip *net.IP) {
	ipdot := strings.Split(ip.String(), ".")
	subnet := strings.Join(ipdot[:len(ipdot)-1], ".") + ".0"
	// log.Printf(subnet)
	cmd := exec.Command("route", "add", "-net",
		subnet, ip.String())

	log.Printf("chu T9 route add -net %s %s", subnet, ip.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("add system route fail: %v, %s", err, string(output))
		panic(fmt.Sprintf("err: %s %s", err, string(output)))
	}
}

func (i *Iface) Name() string {
	return i.ifce.Name()
}

func (i *Iface) Read(pkt PacketIP) (int, error) {
	return i.ifce.Read(pkt)
}

func (i *Iface) Write(pkt PacketIP) (int, error) {
	return i.ifce.Write(pkt)
}
