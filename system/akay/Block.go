package main

import(
  "fmt"
  "net"
  "os"
  "github.com/cilium/ebpf"
  "golang.org/x/sys/unix"
)


var dropMapPath  = "/sys/fs/bpf/map_block"
var dropMap *ebpf.Map

func InitDrop() {
	var err error
	dropMap, err = ebpf.LoadPinnedMap(dropMapPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] Map not loaded: %v\n", err)
		os.Exit(1)
	}
}



const (
	AF_INET  = 2
	AF_INET6 = 10
)

type IPKey struct {
	Addr   [16]byte
}

func GetIPKey(ip string) (IPKey, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
			return IPKey{}, fmt.Errorf("invalid ip")
	}

	key := IPKey{}

	if ipv4 := parsed.To4(); ipv4 != nil {
			//key.Family = AF_INET
			copy(key.Addr[:4], ipv4)
			return key, nil
	}

	ipv6 := parsed.To16()
	if ipv6 != nil {
			//key.Family = AF_INET6
			copy(key.Addr[:16], ipv6)
			return key, nil
	}

	return IPKey{}, fmt.Errorf("invalid ip")
}




func KtimeNS() uint64 {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return 0
	}
	return uint64(ts.Sec)*1e9 + uint64(ts.Nsec)
}


func Drop(ip string, duration uint64){
	write("BLOCKING:",ip)
	key, err := GetIPKey(ip)
	if err != nil {
		write("[X] Ip not readable", ip)
		return 
	}
	var val uint64 = KtimeNS() + duration*1e9
	err = dropMap.Update(key, val, ebpf.UpdateAny)
	if err != nil {
		write(err)
	}else{
		write("[+]BLOCKED")
	}
}

