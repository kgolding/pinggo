package main

import (
	"encoding/binary"
	"net"
	"sort"
)

func GetIPv4NonLocalInterfaces() []string {
	nets := make([]*net.IPNet, 0)

	ifaces, _ := net.Interfaces()
IFaceLoop:
	for _, n := range ifaces {
		if n.Flags&net.FlagLoopback > 0 {
			// ignore loopback
			continue IFaceLoop
		}
		addrs, _ := n.Addrs()
		for _, s := range addrs {
			if ip, ok := s.(*net.IPNet); ok {
				if ip.IP.To4() != nil {
					nets = append(nets, ip)
				}
			}
		}
	}

	// Sort by smallest subnet ascending and IP's desending
	sort.Slice(nets, func(j, k int) bool {
		jj, _ := nets[j].Mask.Size()
		jk, _ := nets[k].Mask.Size()
		if jj == jk { // Same mask
			ji := binary.BigEndian.Uint32(nets[j].IP.To4())
			ki := binary.BigEndian.Uint32(nets[k].IP.To4())
			return ji < ki
		}
		return jj > jk
	})

	// Map nets to strings
	ret := make([]string, len(nets))
	for i, n := range nets {
		ret[i] = n.String()
	}

	return ret
}
