package main

import (
	
	"os"
  "bufio"
  "strings"
	"strconv"
	"os/exec"
	"log"
	

	//"encoding/binary"
	"unsafe"
	"net"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"fmt"
	"slices"
)


var configPATH  string = "bozkurt.cfg"
var kurtPATH    string = "kurt.c"
var bozkurtPATH string = "bozkurt.c"


func ReadFile(path string) string{
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func Bash(command string) (string, int) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()

	if err == nil {
		return string(output), 0
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(output), exitErr.ExitCode()
	}

	// Komut hiç çalıştırılamadı (bash yok, izin yok vb.)
	return string(output), -1
}



var Shell *BashShell


func main() {


	if len(os.Args)<2{
		row("erlik")
		write("  build")
		write("  report")
		write("  delete")
		write("  add map value")
		//write("  enable")
		//write("  disable")
		return
	}

	err := os.Chdir("/web/server/system/test")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if slices.Contains(os.Args, "build") {
		write("[.] build")
		PreBuild()
		Pin()
	}

	if slices.Contains(os.Args, "report") {
		response , _ := Bash("bash command_report.sh")	
		write(response)
		return
	}
	if slices.Contains(os.Args, "delete") {
		response , _ := Bash("bash command_delete.sh")	
		write(response)
	}

	if slices.Contains(os.Args, "add") {
		if( len(os.Args) < 4 ){
			write("[X] erlik add map value")
			return
		}
		ReMaps()
		Add( os.Args[2], os.Args[3] )
		//Add(os.A)
		//response , _ := Bash("bash command_delete.sh")	
		//write(response)
	}
	

	if slices.Contains(os.Args, "list"){
		if( len(os.Args) < 3 ){
			write("[X] erlik list map")
			return
		}
		name := os.Args[2]
		write(Bash("bpftool map dump pinned /sys/fs/bpf/map_"+name))
	}
	


	Shell = NewBashShell()
	defer Shell.Close()


	res, exit := Bash("clang -target bpf -O2 -g -Wall -c bozkurt.c -o bozkurt.o -I/usr/include/x86_64-linux-gnu")
	if(exit!=0){
		write(res)
		write("[X] Error on building bozkurt.c -> bozkurt.o")
		return
	}
	
	//ReadInfo()
}




var MAPS map[string]string
var Codes map[string]string

func PreBuild(){
	code := ReadFile(configPATH)
  ReCode(code)
	kurtData := ReadFile(kurtPATH)
	for key, val := range Codes {
		kurtData = strings.Replace(kurtData, key, val, 1)
	}
	os.WriteFile(bozkurtPATH, []byte(kurtData), 0644)
	write("[+] build success")
}

func ReMaps(){
	
	code := ReadFile(configPATH)

	MAPS := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(code))

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)  // baştaki/sondaki boşlukları temizle
		if line == "" {  // boş satır geç
			continue
		}
		if strings.HasPrefix(line, "#") { 		// # ile başlayan yorum satırı
			continue
		}
		words := strings.Fields(line) 		// space'e göre kelimelere ayır

    if words[0]=="MAPS" {
			if len(words)<3 {
				write("[X] MAPS need 3 parameter, (MAPS TYPE NAME COUNT)")
			}
			kind := words[1]
			name := words[2]
			MAPS[name] = kind
		}
	}
}

func ReCode(code string) {
	hasReturn := 0
  map_codes  := ""
  rules := ""
	MAPS := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(code))

	
	Codes = make(map[string]string)
	Codes["//@@IPv4CDIR@@"]  = `
struct IPv4CDIR{
	__u32 prefixlen;
	__u32 addr;
};
`

	Codes["//@@IPv6CDIR@@"]  = `
struct IPv6CDIR{
	__u32 prefixlen;
	union {
		__u8 bytes[16];
		__u64 IPv6[2];
	} addr;
};
`


	Codes["//@@ParsePacket@@"] = `
struct IPKey {
	__u8 Family;
	__u8 Pad[3];
	union {
		__be32 IPv4;
		__u64 IPv6[2];
	} Addr;
};

struct TCPInfo {
    __u16 SrcPort;
    __u16 DstPort;
    __u8 Flags;
    __u8 DoL;
};

struct PacketInfo {
	struct IPKey IP;
	__be16 Port;
	__u8 Protocol;
	__u8 Pad;
	__u32 Length;
	__u8 IsFragmented;
	struct TCPInfo TCP;
};	
`



	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)  // baştaki/sondaki boşlukları temizle
		if line == "" {  // boş satır geç
			continue
		}
		if strings.HasPrefix(line, "#") { 		// # ile başlayan yorum satırı
			continue
		}
		words := strings.Fields(line) 		// space'e göre kelimelere ayır

    if words[0]=="MAPS" {
			if len(words)<3 {
				write("[X] MAPS need 3 parameter, (MAPS TYPE NAME COUNT)")
			}
			write("[.]",line)
			kind := words[1]
			name := words[2]
			count := "1000000"
			if len(words)>3{
				count = words[3]
			}
			MAPS[name] = kind
			if kind=="IP_DURATION" {
				map_codes += `
//# `+line+`
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_HASH);        
	/* Maksimum Data */     __uint(max_entries,  `+count+`);         
	/* Anahtar       */     __type(key,          struct IPKey);              
	/* Değer         */     __type(value,        __u64);  
} `+name+` SEC(".maps");
`
			}

			if kind == "IP"{
				map_codes += `
//# `+line+`
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_HASH);        
	/* Maksimum Data */     __uint(max_entries,  `+count+`);         
	/* Anahtar       */     __type(key,          struct IPKey);              
	/* Değer         */     __type(value,        __u8);  
} `+name+` SEC(".maps");
`				
			}


			
			if kind == "CDIRv4"{
				map_codes += `
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_LPM_TRIE);        
	/* Maksimum Data */     __uint(max_entries,  `+count+`);         
	/* Anahtar       */     __type(key,          struct IPv4CDIR);              
	/* Değer         */     __type(value,        __u8);  
	/* ALLOC         */     __uint(map_flags,    BPF_F_NO_PREALLOC);
} `+name+` SEC(".maps");
`				
			}

			
			if kind == "CDIRv6"{
				map_codes += `
//# `+line+`
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_LPM_TRIE);        
	/* Maksimum Data */     __uint(max_entries,  `+count+`);         
	/* Anahtar       */     __type(key,          struct IPv6CDIR);
	/* Değer         */     __type(value,        __u8);  
	/* ALLOC         */     __uint(map_flags,    BPF_F_NO_PREALLOC);
} `+name+` SEC(".maps");
`				
			}


			
    }

		


		// DROP 
		if words[0]=="DROP"{
			// DROP black;
			if len(words)== 2{
				name := words[1]
				if MAPS[name] == "IP_DURATION"{
					rules += `
//# `+line+`
__u64 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &packet.IP);
if (IsFound_`+name+`) {
	__u64 now = bpf_ktime_get_ns();
	if (*IsFound_`+name+` > now){
		send_log(0, &packet);
		return XDP_DROP;
	}else{
		bpf_map_delete_elem(&`+name+`, &packet.IP);
	}
}
`

				}

				if MAPS[name] == "IP"{
					rules += `
//# `+line+`
__u64 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &packet.IP);
if (IsFound_`+name+`) {
	send_log(0, &packet);
	return XDP_DROP;
}
`
				}


				if MAPS[name] == "CDIRv4" {
					rules += `
//# `+line+`
if (packet.IP.Family == 2) {
	struct IPv4CDIR Key_`+name+` = { .prefixlen = 32, .addr      = packet.IP.Addr.IPv4 };
	__u8 *IsFound_`+name+` = bpf_map_lookup_elem(&asn_ipv4_map, &Key_`+name+`);
	if (IsFound_`+name+`) {
		//log_asn_drop(packet.Length);
		send_log(0, &packet);
		return XDP_DROP;
	}
}
`
				}

				if MAPS[name] == "CDIRv6" {
					rules += `
//# `+line+`
if (packet.IP.Family == 10) {
	struct IPv4CDIR Key_`+name+` = { .prefixlen = 32, .addr      = packet.IP.Addr.IPv4 };
	__u8 *IsFound_`+name+` = bpf_map_lookup_elem(&asn_ipv4_map, &Key_`+name+`);
	if (IsFound_`+name+`) {
		//log_asn_drop(packet.Length);
		send_log(0, &packet);
		return XDP_DROP;
	}
}
`
				}

			// DROP;
			}else if(len(words)==1){
				rules += `
//# `+line+`
send_log(0, &packet);
return XDP_DROP;`
				hasReturn = 1
			}
		}





		if words[0]=="PASS"{
			// DROP black;
			if len(words)== 2{
				name := words[1]
				if MAPS[name] == "IP_DURATION"{
					rules += `
//# `+line+`
__u64 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &packet.IP);
if (IsFound_`+name+`) {
	__u64 now = bpf_ktime_get_ns();
	if (*IsFound_`+name+` < now){
		send_log(0, &packet);
		return XDP_PASS;
	}else{
		bpf_map_delete_elem(&`+name+`, &packet.IP);
	}
}
`

				}

				if MAPS[name] == "IP"{
					rules += `
//# `+line+`
__u64 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &packet.IP);
if (IsFound_`+name+`) {
	send_log(0, &packet);
	return XDP_PASS;
}
`
				}

			// DROP;
			}else if(len(words)==1){
				rules += `
//# `+line+`
send_log(1, &packet);
return XDP_PASS;
`
				hasReturn = 1
			}
		}

		if words[0]=="GUARD" && words[1]=="SYNFloodAttack"{
			rules += GuardSYNFloodAttack(line,words)
		}
		if words[0]=="GUARD" && words[1]=="FragmentAttack"{
			rules += GuardFragmentAttack(line,words)
		}
		// test için basalım
		//fmt.Println(words)
	}

	// FOr Test

	//write("//=========== MAPS ===========")
	//write ( maps )
	//write("//=========== RULES ===========")
	//write ( rules )

	Codes["//@@MAPS@@"] = map_codes

	Codes["//@@RULES@@"] = rules

	if(hasReturn==0){
		write("[X] Program must need at last DROP or PASS")
		os.Exit(1)
	}


}



func GuardFragmentAttack(line string,words []string) string{

return `
//# `+line+`
if (packet.IsFragmented) {
	log_frag_drop(packet.Length);
	send_log(0, &packet);
	return XDP_DROP;
}
`

}


func GuardSYNFloodAttack(line string,words []string) string{
	if( len(words)!=4 ){
		write("[X] Guard SYNFloodAttack [IP_DURATION] [TIME]")
		os.Exit(1)
	}
	name := words[2]
	if MAPS[name] != "IP_DURATION"{
		write("[X] Guard SYNFloodAttack [IP_DURATION]: 3. parameters must be IP_DURATION map")
		os.Exit(1)
	}

	duration := words[3]
	seconds, err := strconv.ParseUint(duration, 10, 64)
	if err != nil {
		panic(err)
	}
	expireNS := seconds * 1000000000;
	expireNSStr := strconv.FormatUint(expireNS, 10)

	return `
//# `+line+`
if (packet.Protocol == IPPROTO_TCP) {
		if (!CheckTCPFlags(&packet.TCP)) {
				log_tcp_anomaly_drop(packet.Length);
				send_log(0, &packet); // Test için anında logla
				return XDP_DROP; // Geçersiz TCP Bayrak Kombinasyonu (Nmap Stealth Scan vs.)
		}

		// SYN Flood Hız Sınırı (Rate Limit)
		if (packet.TCP.Flags & 0x02) { // 0x02 == SYN
				__u32 syn_key = 0;
				__u64 *global_syn = bpf_map_lookup_elem(&global_syn_stats, &syn_key);
				if (global_syn) {
						__sync_fetch_and_add(global_syn, 1);
						
						// Panic Mode: If Global SYN > 3000 per second
						// (bozkurt_daemon will reset this to 0 every second)
						if (*global_syn > 3000) {
								// Check for MSS Option (Basic OS Fingerprint)
								// If the header length is just 20 (no options), drop.
								if ((packet.TCP.DoL >> 4) < 6) { 
										return XDP_DROP; // Drop spoofed/dumb SYN
								}
								
								// Skip LRU map update to save CPU in panic mode
								// Pass to Kernel SYN Cookies
								goto skip_lru;
						}
				}

				struct syn_tracker_value *syn_val = bpf_map_lookup_elem(&syn_rate_limit, &packet.IP);
				__u64 now = bpf_ktime_get_ns();
				if (syn_val) {
						if (now - syn_val->first_seen > 1000000000ULL) { // 1 Saniye dolduysa sıfırla
								syn_val->count = 1;
								syn_val->first_seen = now;
						} else {
								syn_val->count++;
								if (syn_val->count > 10) { // Saniyede 10 SYN'den fazlaysa 1 SAAT BANLA!
										__u64 guard_syn_flood_attack_time_out= now + `+expireNSStr+`;
										bpf_map_update_elem(&`+name+`, &packet.IP, &guard_syn_flood_attack_time_out, BPF_ANY);
										
										log_syn_flood_drop(packet.Length);
										send_log(0, &packet); // Test için anında logla
										return XDP_DROP;
								}
						}
				} else {
						struct syn_tracker_value new_val = {};
						new_val.count = 1;
						new_val.first_seen = now;
						bpf_map_update_elem(&syn_rate_limit, &packet.IP, &new_val, BPF_ANY);
				}
skip_lru:
				;
		}
}
	
`
	
}














func ReadInfo(){
	spec, err := ebpf.LoadCollectionSpec("bozkurt.o")
	if err != nil {
		fmt.Printf("load spec: %v", err)
	}

	fmt.Println("Programs:")
	for name, prog := range spec.Programs {
		fmt.Printf("- %s\n", name)
		fmt.Printf("  Type: %s\n", prog.Type)
		fmt.Printf("  Section: %s\n", prog.SectionName)
		fmt.Printf("  License: %s\n", prog.License)
	}

	fmt.Println("\nMaps:")
	for name, m := range spec.Maps {
		fmt.Printf("- %s\n", name)
		fmt.Printf("  Type: %s\n", m.Type)
		fmt.Printf("  KeySize: %d\n", m.KeySize)
		fmt.Printf("  ValueSize: %d\n", m.ValueSize)
		fmt.Printf("  MaxEntries: %d\n", m.MaxEntries)
	}
}






const (
	AF_INET  = 2
	AF_INET6 = 10
)

type IPKey struct {
	Family uint8
	Pad    [3]uint8
	Addr   [16]byte
	_      [4]byte

}

func GetIPKey(ip string) (IPKey, error) {
    parsed := net.ParseIP(ip)
    if parsed == nil {
        return IPKey{}, fmt.Errorf("invalid ip")
    }

    key := IPKey{}

    if ipv4 := parsed.To4(); ipv4 != nil {
        key.Family = AF_INET
        copy(key.Addr[:4], ipv4)
        return key, nil
    }

    ipv6 := parsed.To16()
    if ipv6 != nil {
        key.Family = AF_INET6
        copy(key.Addr[:16], ipv6)
        return key, nil
    }

    return IPKey{}, fmt.Errorf("invalid ip")
}

func Add(name string, value string){
	m, err := ebpf.LoadPinnedMap("/sys/fs/bpf/map_"+name, nil)
	if err != nil {
		log.Fatalf("[x] load pinned map: %v", err)
	}
	defer m.Close()

	
	key, err := GetIPKey(value)
	if err != nil {
		return 
	}
	write(key)
	fmt.Println(unsafe.Sizeof(IPKey{}))

	fmt.Printf("%+v\n", key)
	fmt.Printf("%v.%v.%v.%v\n",
			key.Addr[0],
			key.Addr[1],
			key.Addr[2],
			key.Addr[3],
	)

	var val uint64 = 1



	b := (*[20]byte)(unsafe.Pointer(&key))

	for i := range b {
			fmt.Printf("%02x ", b[i])
	}
	fmt.Println()


	err = m.Update(key, val, ebpf.UpdateNoExist)
	if err != nil {
			write(err)
	}

	/*
	if err := m.Put(key, val); err != nil {
		log.Fatalf("put: %v", err)
	}
	*/

	log.Println("Map güncellendi.")
}





func IFACE() string {
	out, _ := exec.Command("sh", "-c", "ip route show default | awk '{print $5}'").Output()
	return strings.TrimSpace(string(out))
}


func Pin() {

	iface, err := net.InterfaceByName(IFACE())
	if err != nil {
		write("[X]", err)
		return
	}


	spec, err := ebpf.LoadCollectionSpec("bozkurt.o")
	if err != nil {
		write("[X] LoadCollectionSpec:", err)
		return
	}


	// BPF objelerini oluştur
	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		write("[X] NewCollection:", err)
		return
	}



	fmt.Println("\nLoaded maps:", len(coll.Maps))
	for name := range coll.Maps {
		fmt.Println("MAP:", name)
	}



	
	// Map pinleme
	fmt.Println("\n=== Pin Maps ===")
	for name, m := range coll.Maps {
		if( name==".rodata" ){ continue }
		path := "/sys/fs/bpf/map_" + name
		// eski pin varsa kaldır
		os.Remove(path)
		err := m.Pin(path)

		if err != nil {
			write("[X] map pin:", name, err)
			continue
		}
		write("[+] map pinned:", path)
	}


	for name := range coll.Maps {
		if( name==".rodata" ){ continue }
		path := "/sys/fs/bpf/map_" + name
		err := coll.Maps[name].Pin(path)
		fmt.Println("PIN:",name,"PATH:",path,"ERR:",err,)
	}


	
	// Program al
	prog, ok := coll.Programs["Bozkurt"]
	if !ok {
		write("[X] Bozkurt program bulunamadı")
		return
	}

	// Program pinleme
	progPath := "/sys/fs/bpf/bozkurt_program"
	os.Remove(progPath)
	err = prog.Pin(progPath)
	if err != nil {
		write("[X] program pin:", err)
		return
	}
	write("[+] program pinned:", progPath)

	// XDP attach
	l, err := link.AttachXDP(link.XDPOptions{
		Program:   prog,
		Interface: iface.Index,
	})
	if err != nil {
		write("[X] XDP attach:", err)
		return
	}


	//	Link pinleme
	linkPath := "/sys/fs/bpf/bozkurt_link"
	err = l.Pin(linkPath)
	if err != nil {
		write("[X] link pin:", err)
		return
	}


	write("[+] link pinned:", linkPath)



	
	fmt.Println("\n=== Maps ===")
	for name, m := range spec.Maps {
		fmt.Printf(
			"MAP %-20s %-15v Struct(%3d,%3d) MaxSize=%-8d Flags=%v\n",
			name,m.Type,m.KeySize,m.ValueSize,m.MaxEntries,m.Flags,
		)
	}


	fmt.Println("\n=== Programs ===")
	for name, p := range spec.Programs {
		fmt.Printf(
			"PROG %-20s Type=%-10v License=%-10s Instructions=%d\n",
			name,p.Type,p.License,len(p.Instructions),
		)
	}


	write("[+] bozkurt aktif")

	
}