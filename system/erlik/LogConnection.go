package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"
)

type TCPInfo struct {
	SrcPort uint16
	DstPort uint16
	Flags   uint8
	DoL     uint8
}

type PacketInfo struct {
	IP [16]byte
	InPort uint16
	RePort uint16
	Protocol uint8
	Length uint32
	IsFragmented uint8
	Family       uint8
	TCP TCPInfo
	Info string
}

func LogConnections(name string) {
	// Pinlenmiş RingBuf map'i aç.
	m, err := ebpf.LoadPinnedMap("/sys/fs/bpf/map_LOGS", nil)
	if err != nil {
		fmt.Printf("[X] load pinned map: %v\n", err)
		return
	}
	defer m.Close()

	// Ring buffer reader.
	reader, err := ringbuf.NewReader(m)
	if err != nil {
		fmt.Printf("[X] create ringbuf reader: %v\n", err)
		return
	}
	defer reader.Close()

	log.Println("Listening on /sys/fs/bpf/map_LOGS")

	for {
		record, err := reader.Read()
		if err != nil {
			if err == ringbuf.ErrClosed {
				return
			}
			log.Printf("read ringbuf: %v", err)
			continue
		}

		info, err := parsePacketInfo(record.RawSample)
		if err != nil {
			log.Printf("parse packet: %v", err)
			continue
		}
		if name=="" || info.Info == name{
			printPacket(info)
		}
	}
}


func protocolName(proto uint8) string {
	switch proto {
	case 1:
		return "ICMP"
	case 6:
		return "TCP"
	case 17:
		return "UDP"
	case 41:
		return "IPv6"
	case 47:
		return "GRE"
	case 58:
		return "ICMPv6"
	default:
		return fmt.Sprintf("%d", proto)
	}
}

func parsePacketInfo(b []byte) (*PacketInfo, error) {

	if len(b) < 40 {
		return nil, fmt.Errorf("packet too small: %d bytes", len(b))
	}

	var p PacketInfo
	copy(p.IP[:], b[0:16])
	p.InPort = binary.BigEndian.Uint16(b[16:18])
	p.RePort = binary.BigEndian.Uint16(b[18:20])
	p.Protocol = b[20]
	//p.Pad = b[21]

	p.Length = binary.LittleEndian.Uint32(b[24:28])
	p.IsFragmented = b[28]
	p.Family = b[29]
	
	p.TCP.SrcPort = binary.BigEndian.Uint16(b[30:32])
	p.TCP.DstPort = binary.BigEndian.Uint16(b[32:34])
	p.TCP.Flags = b[34]
	p.TCP.DoL = b[35]

	p.Info = string([]byte{b[39], b[38], b[37], b[36]})
	p.Info = strings.TrimRight(p.Info, "\x00")
	return &p, nil
}

func printPacket(p *PacketInfo) {
	var ip string

	switch p.Family {

	case 2:
		ip = net.IP(p.IP[0:4]).String()

	case 10:
		ip = net.IP(p.IP[0:16]).String()

	default:
		ip = fmt.Sprintf("unknown-family(%d)", p.Family)
	}

	fmt.Printf(
		"%-4s %7s:%5d -> %-5d %5db %-39s Fd=%d "+
			"Flags=0x%02x DoL=%d\n",

		p.Info,
		protocolName(p.Protocol),
		p.RePort,
		p.InPort,
		p.Length,
		ip,
		p.IsFragmented,
		//p.TCP.SrcPort,
		//p.TCP.DstPort,
		p.TCP.Flags,
		p.TCP.DoL,
	)

}
