package main

import (
	"net"
	"sync"
	"time"
)

var TCPPorts = map[int]string{
	21:  "FTP",
	22:  "SSH",
	23:  "TELNET",
	80:  "HTTP",
	123: "NTP",
	443: "HTTPS",
}

type IPInfo struct {
	Active     bool
	LastChange time.Time
	IP         net.IP
	// TCPPorts   []int
	sync.Mutex
}

// func (i *IPInfo) AddTCPPort(port int) {
// 	i.Lock()
// 	i.TCPPorts = append(i.TCPPorts, port)
// 	sort.Slice(i.TCPPorts, func(a, b int) bool {
// 		return i.TCPPorts[a] < i.TCPPorts[b]
// 	})
// 	i.Unlock()
// }

func (i *IPInfo) String() string {
	s := i.IP.String()
	// for _, port := range i.TCPPorts {
	// 	if n, ok := TCPPorts[port]; ok {
	// 		s += "\t" + n
	// 	} else {
	// 		s += fmt.Sprintf("\t%d", port)
	// 	}
	// }
	return s
}
