package main

import (
	"net"
)

func GetIPv4NonLocalInterfaces() []string {
	ret := make([]string, 0)

	ifaces, _ := net.Interfaces()
IFaceLoop:
	for _, n := range ifaces {
		if n.Flags&net.FlagLoopback > 0 {
			continue IFaceLoop
		}

		addrs, _ := n.Addrs()
		for _, s := range addrs {
			if ip, ok := s.(*net.IPNet); ok {
				if ip.IP.To4() != nil {
					ret = append(ret, s.String())
				}
			}
		}
	}

	return ret
}
