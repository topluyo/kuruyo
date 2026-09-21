package main
import (
  "os"
  "fmt"
  "net"
	"github.com/cilium/ebpf"
	"golang.org/x/sys/unix"
)


var BOZKURT_dropMap *ebpf.Map
var BOZKURT_passMap *ebpf.Map


type BOZKURT_IPKey struct {
	Addr   [16]byte
}


func BOZKURT_GetIPKey(ip string) (BOZKURT_IPKey, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return BOZKURT_IPKey{}, fmt.Errorf("invalid ip")
	}
	key := BOZKURT_IPKey{}
	if ipv4 := parsed.To4(); ipv4 != nil {
		copy(key.Addr[:4], ipv4)
		return key, nil
	}
	ipv6 := parsed.To16()
	if ipv6 != nil {
		copy(key.Addr[:16], ipv6)
		return key, nil
	}
	return BOZKURT_IPKey{}, fmt.Errorf("invalid ip")
}


func BOZKURT_GetIPKeyVal() uint64 {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return 0
	}
	return uint64(ts.Sec)*1e9 + uint64(ts.Nsec)
}


func BOZKURT_Init(BOZKURT_dropMapPath string){
	var err error
	BOZKURT_dropMap, err = ebpf.LoadPinnedMap(BOZKURT_dropMapPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] Map not loaded: %v\n", err)
		os.Exit(1)
	}
}


func BOZKURT_Drop(ip string, duration uint64){
	write("BLOCKING:",ip)
	key, err := BOZKURT_GetIPKey(ip)
	if err != nil {
		write("[X] Ip not readable", ip)
		return 
	}
	var val uint64 = BOZKURT_GetIPKeyVal() + duration*1e9
	err = BOZKURT_dropMap.Update(key, val, ebpf.UpdateAny)
	if err != nil {
		write(err)
	}else{
		write("[+]BLOCKED")
	}
}


func BOZKURT_Pass(ip string, duration uint64){
	write("BLOCKING:",ip)
	key, err := BOZKURT_GetIPKey(ip)
	if err != nil {
		write("[X] Ip not readable", ip)
		return 
	}
	var val uint64 = BOZKURT_GetIPKeyVal() + duration*1e9
	err = BOZKURT_passMap.Update(key, val, ebpf.UpdateAny)
	if err != nil {
		write(err)
	}else{
		write("[+]BLOCKED")
	}
}

