package main

import (
	
	"os"
	"bufio"
	"strings"
	"strconv"
	"os/exec"
	"log"
	"encoding/binary"
	
	"time"
	"regexp"

	//"encoding/binary"
	//"unsafe"
	"net"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"fmt"
	"slices"
	"golang.org/x/sys/unix"
	"github.com/godbus/dbus/v5"

)

var erlikPATH   string = "/usr/local/bin/erlik"
var workingPATH string = "/web/system/erlik"
var configPATH  string = "bozkurt.cfg"
var kurtPATH    string = "kurt.c"
var bozkurtPATH string = "bozkurt.c"
const SYSTEMD_PATH     = "/etc/systemd/system/"
const reportMapPath    = "/sys/fs/bpf/map_REPORT"
var Systemd  dbus.BusObject

func ReadFile(path string) string{
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func WriteFile(path, data string){
	os.WriteFile(path, []byte(data), 0644)
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


func InitSystemD(){
	conn, err := dbus.SystemBus()
	if err != nil {
		panic(err)
	}
	//defer conn.Close()
	Systemd = conn.Object("org.freedesktop.systemd1","/org/freedesktop/systemd1")
}

func main() {


	if len(os.Args)<2{
		row("erlik")
		write("\nBUILD ACTIONS")
		write("  test                     - test program")
		write("  build                    - build program and map")
		write("  asm                      - writes asm code ")
		write("  delete                   - delete program and maps")
		write("[X] disable                  - delete program")

		write("\nRUNNING ACTIONS")
		write("  info                     - report builded program and maps")
		write("  report                   - last day info")
		write("  log [RESP]               - log any request")

		write("\nMAP ACTIONS")
		write("  add map value [second]   - add item into map")
		write("  list [map]               - list maps")
		write("  clear [map]              - clear maps")
		write("  count [map]              - count maps")
		write("  mode  [code]             - set mode")

		
		write("\nSERVICE ACTIONS")
		write("  loop                     - updates Period, Time, Uniq")
		write("  start                    - create and start service")
		//write("  [X] sistem ortasında veriye ulaşılamayınca tekrar denemiyor")
		write("  stop                     - delete and stop service")
		write("  status                   - listen service journal")
		
		
		return
	}

	err := os.Chdir(workingPATH)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if slices.Contains(os.Args, "test") {
		write("[.] test")
		PreBuild()
		
		//res, exit := Bash("clang -target bpf -O2 -g -Wall -c bozkurt.c -o bozkurt.o -I/usr/include/x86_64-linux-gnu")
		//res, exit := Bash("clang -O3 -target bpf -mcpu=v3 -flto -fno-stack-protector -Wall -I/usr/include/x86_64-linux-gnu -c bozkurt.c -o bozkurt.o")
		res, exit := Bash("clang -O2 -target bpf -mcpu=v3 -g -Wall -D__TARGET_ARCH_x86 -c bozkurt.c -o bozkurt.o -I/usr/include/x86_64-linux-gnu")
		if(exit!=0){
			write(res)
			write("[X] Error on building bozkurt.c -> bozkurt.o")
		}		
		write("[+] test success")
		return
	}

	if slices.Contains(os.Args, "build") {
		write("[.] build")
		PreBuild()
		res, exit := Bash("clang -O2 -target bpf -mcpu=v3 -g -Wall -D__TARGET_ARCH_x86_64 -c bozkurt.c -o bozkurt.o -I/usr/include/x86_64-linux-gnu")
		if(exit!=0){
			write(res)
			write("[X] Error on building bozkurt.c -> bozkurt.o")
			return
		}		
		Pin()
	}

	if slices.Contains(os.Args, "info") {
		response , _ := Bash("bash command_info.sh")	
		write(response)
		return
	}

	if slices.Contains(os.Args, "report") {
		reportMap, err := ebpf.LoadPinnedMap(reportMapPath, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[X] Map not loaded: %v\n", err)
			os.Exit(1)
		}
		defer reportMap.Close()
		numCPU, err := ebpf.PossibleCPU()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[X] ebpf.PossibleCPU() %v\n", err)
			return
		}

		reports := make([]Report, numCPU)

		
		now    := time.Now()
		period := uint32(now.Hour()*60 + now.Minute())
		
		for i:=uint32(period+1) ; i<1440+period ; i++{
			index := uint32(i) % 1440
			if err := reportMap.Lookup(&index, &reports); err != nil {
				fmt.Fprintf(os.Stderr, "[X] key not found in REPORT %v\n", err)
				return 
			}else{
				//fmt.Printf("%02d:%02d %8d ✔ %8d ✘ %9d KB %8d IP\n",index/60,index%60, reports[0].Pass, reports[0].Drop, reports[0].Byte/1024, reports[0].Uniq )
				fmt.Printf("%02d:%02d %8d PASS %8d DROP %9d KB %8d IP\n",index/60,index%60, reports[0].Pass, reports[0].Drop, reports[0].Byte/1024, reports[0].Uniq )
			}
		}

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
		duration := uint64(60);
		if( len(os.Args)==5 ){
			duration = uint64(ToNumber(os.Args[4]))
		}
		Add( os.Args[2], os.Args[3], duration, true )
		
	}

	
	if slices.Contains(os.Args, "mode") {
		if( len(os.Args) < 2 ){
			write("[X] erlik set MODE")
			return
		}
		ReMaps()
		SetMode( os.Args[2] )
		
	}
	
	if slices.Contains(os.Args, "clear"){
		if( len(os.Args) < 3 ){
			write("[X] erlik clear map")
			return
		}
		Clear(os.Args[2])
	}

	if slices.Contains(os.Args, "count"){
		if( len(os.Args) < 3 ){
			write("[X] erlik count map")
			return
		}
		Count(os.Args[2])
	}

	if slices.Contains(os.Args, "list"){
		if( len(os.Args) < 3 ){
			p, err := ebpf.LoadPinnedProgram("/sys/fs/bpf/bozkurt_program", nil)
			if err != nil {
					log.Fatal(err)
			}
			defer p.Close()

			info, err := p.Info()
			if err != nil {
					log.Fatal(err)
			}


			mapIDs, ok := info.MapIDs()
			if !ok {
					fmt.Println("Program için Map ID bilgisi mevcut değil")
					return
			}

			for _, mapID := range mapIDs {
					m, err := ebpf.NewMapFromID(mapID)
					if err != nil {
							fmt.Printf("Map ID %d açılamadı: %v\n", mapID, err)
							continue
					}

					mapInfo, err := m.Info()
					if err != nil {
							fmt.Printf("Map ID %d info alınamadı: %v\n", mapID, err)
							m.Close()
							continue
					}

					fmt.Printf(
							"  %-20s %-6d %s\n",
							mapInfo.Name,
							mapID,
							mapInfo.Type,
					)

					m.Close()
			}




			return
		}
		name := os.Args[2]
		write(Bash("bpftool map dump pinned /sys/fs/bpf/map_"+name))
	}


	if slices.Contains(os.Args, "start") {
		file := CreateServiceFileContent(erlikPATH + " loop",workingPATH)
		WriteFile(SYSTEMD_PATH+"/erlik.service",file)
		InitSystemD()
		CommandDaemonReload()
		CommandStopService("erlik")
		CommandStartService("erlik")
		CommandEnableService("erlik")
	}else if slices.Contains(os.Args, "stop"){
		InitSystemD()
		CommandStopService("erlik")
		CommandDisableService("erlik")
		CommandDaemonReload()
		os.Remove(SYSTEMD_PATH+"/erlik.service")
	}

	if slices.Contains(os.Args, "loop") {
		ReMaps()
		CreateMapServer()
		MinuteLoop()
	}



	if slices.Contains(os.Args, "asm") {
		p, err := ebpf.LoadPinnedProgram("/sys/fs/bpf/bozkurt_program",nil,)
		if err != nil {
			write("[X] /sys/fs/bpf/bozkurt_program not found",err)
			os.Exit(1)
		}
		defer p.Close()

		info, err := p.Info()
		if err != nil {
			panic(err)
		}

		id, ok := info.ID()
		if !ok {
			panic("program ID alınamadı")
		}

		idStr := fmt.Sprintf("%d", id)


		response , _ := Bash("bpftool prog dump xlated id "+idStr+" opcodes")	
		write(response)
		write("[+] bpftool prog dump xlated id "+idStr+" opcodes")
		//LogConnections()
		return
	}

	if slices.Contains(os.Args, "disable") {
		response , _ := Bash("bash command_delete_programs.sh")
		write(response)
	}
	

	if slices.Contains(os.Args, "log"){
		if len(os.Args)==3{
			LogConnections(os.Args[2])
		}else{
			LogConnections("")
		}
	}

	Shell = NewBashShell()
	if slices.Contains(os.Args, "status"){
		Shell.Run("journalctl -f -u erlik -o cat -n 360")
	}




	
	defer Shell.Close()


	
	//ReadInfo()
}


func CreateServiceFileContent(command, workdir string) string {
	return fmt.Sprintf(`[Unit]
After=network.target

[Service]
ExecStart=%s
Restart=always
User=root
WorkingDirectory=%s

[Install]
WantedBy=multi-user.target
`, command, workdir)
}






var MAPS map[string]string
var ENABLES map[string]string
var Codes map[string]string
var ADDS map[string][]string
var CLEARS map[string]int

func PreBuild(){
	code := ReadFile(configPATH)
	ReMaps()
  ReCode(code)
	kurtData := ReadFile(kurtPATH)
	for key, val := range Codes {
		kurtData = strings.Replace(kurtData, key, val, 1)
	}
	WriteFile(bozkurtPATH, kurtData)
	write("[+] build success")
}

func ReMaps(){
	
	code := ReadFile(configPATH)

	MAPS    = make(map[string]string)
	ENABLES = make(map[string]string)
	ADDS    = make(map[string][]string)
	CLEARS  = make(map[string]int)
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

    if words[0]=="MAP" {
			if len(words)<3 {
				write("[X] MAPS need 3 parameter, (MAPS TYPE NAME COUNT)")
			}
			kind := words[1]
			name := words[2]
			MAPS[name] = kind
		}
		if words[0]=="ENABLE" {
			ENABLES[words[1]] = "1"
		}
		if words[0]=="ADD"{
			if _, ok := ADDS[words[1]]; !ok {
				ADDS[words[1]] = make([]string,0)
			}
			for _,val := range strings.Split(words[2],","){
				ADDS[words[1]] = append(ADDS[words[1]],val)
			}
		}
		if words[0]=="CLEAR"{
			CLEARS[words[1]]=1
		}
	}
	
}


func StringToHex4(s string) string {
	b := [4]byte{' ', ' ', ' ', ' '}
	copy(b[:], s)

	return fmt.Sprintf("0x%02X%02X%02X%02X",b[0], b[1], b[2], b[3])
}


func convertIPs(text string) string {
	re := regexp.MustCompile(`IP\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		m := re.FindStringSubmatch(match)
		var ip uint32
		for i := 1; i <= 4; i++ {
			n, err := strconv.ParseUint(m[i], 10, 8)
			if err != nil {
				return match // geçersizse olduğu gibi bırak
			}
			ip = (ip << 8) | uint32(n)
		}

		return fmt.Sprintf("0x%08X", ip)
	})
}

func convertPORTs(text string) string {
	re := regexp.MustCompile(`PORT\.(\d+)`)

	return re.ReplaceAllStringFunc(text, func(match string) string {
		m := re.FindStringSubmatch(match)

		port, err := strconv.ParseUint(m[1], 10, 16)
		if err != nil {
			return match // geçersizse olduğu gibi bırak
		}

		// bpf_htons karşılığı: 16-bit byte swap
		port = ((port & 0xFF) << 8) | ((port >> 8) & 0xFF)

		return fmt.Sprintf("0x%04X", port)
	})
}

func RecodeHelper(state string, path string) string{
	resp := ""

	//! test here
	if _,ok:=ENABLES["UNIQ_IP"]; ok { resp += "\t\tUniq_IP(&packet);\n" }
	if _,ok:=ENABLES["UNIQ"]; ok { resp += "\t\tUniq(&packet);\n" }
	if _,ok:=ENABLES["STAT"]; ok { resp += "\t\tStat(&packet, "+state+");\n" }
	if(path!=""){
		resp += "\t\tLog(&packet, "+StringToHex4(path)+", "+state+");\n"
	}
	return resp
}
func ReCode(code string) {
	hasReturn := 0
  map_codes  := ""
  rules := ""
	//MAPS := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(code))
	Codes = make(map[string]string)

	index := 0
	for scanner.Scan() {
		index++
		line := scanner.Text()
		line = strings.TrimSpace(line)  // baştaki/sondaki boşlukları temizle
		if line == "" {  // boş satır geç
			continue
		}
		if strings.HasPrefix(line, "#") { 		// # ile başlayan yorum satırı
			continue
		}

		line = convertIPs(line)
		line = convertPORTs(line)

		var path string = ""
		re := regexp.MustCompile(`"([^"]+)"`)
		match := re.FindStringSubmatch(line)
		if len(match) > 1 {
			path = match[1]
			line = re.ReplaceAllString(line, "")
		}
		words := strings.Fields(line)


		
		if words[0]=="ENABLE" {
			ENABLES[words[1]] = "1"
		}
    if words[0]=="MAP" {
			if len(words)<3 {
				write("[X] MAP need 3 parameter, (MAP TYPE NAME COUNT)")
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
//# `+ToString(index)+":"+line+`
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
//# `+ToString(index)+":"+line+`
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_HASH);        
	/* Maksimum Data */     __uint(max_entries,  `+count+`);         
	/* Anahtar       */     __type(key,          struct IPKey);              
	/* Değer         */     __type(value,        __u8);  
} `+name+` SEC(".maps");
`
			}


			if kind == "INPORT" || kind == "OTPORT" {
				map_codes += `
//# `+ToString(index)+":"+line+`
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_ARRAY);        
	/* Maksimum Data */     __uint(max_entries,  65536);         
	/* Anahtar       */     __type(key,          __u32);
	/* Değer         */     __type(value,        __u8);  
} `+name+` SEC(".maps");
`				
			}


			
			if kind == "CIDRv4"{
				map_codes += `
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_LPM_TRIE);        
	/* Maksimum Data */     __uint(max_entries,  `+count+`);         
	/* Anahtar       */     __type(key,          struct IPv4CIDR);              
	/* Değer         */     __type(value,        __u8);  
	/* ALLOC         */     __uint(map_flags,    BPF_F_NO_PREALLOC);
} `+name+` SEC(".maps");
`				
			}

			
			if kind == "CIDRv6"{
				map_codes += `
//# `+ToString(index)+":"+line+`
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_LPM_TRIE);        
	/* Maksimum Data */     __uint(max_entries,  `+count+`);         
	/* Anahtar       */     __type(key,          struct IPv6CIDR);
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
//# `+ToString(index)+":"+line+`
__u64 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &packet.IP);
if (IsFound_`+name+`) {
	__u64 now = bpf_ktime_get_ns();
	if (*IsFound_`+name+` > now){
`+RecodeHelper("0",path)+`
		return XDP_DROP;
	}else{
		bpf_map_delete_elem(&`+name+`, &packet.IP);
	}
}
`

				}

				if MAPS[name] == "IP"{
					rules += `
//# `+ToString(index)+":"+line+`
__u64 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &packet.IP);
if (IsFound_`+name+`) {
`+RecodeHelper("0",path)+`
	return XDP_DROP;
}
`
				}

				if MAPS[name] == "OTPORT"{
					rules += `
//# `+ToString(index)+":"+line+`
__u32 Port_`+name+` = __builtin_bswap16(packet.OtPort);
__u8 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &Port_`+name+`);
if (IsFound_`+name+` && *IsFound_`+name+`==1) {
`+RecodeHelper("0",path)+`
	return XDP_DROP;
}
`
				}

				if MAPS[name] == "INPORT"{
					rules += `
//# `+ToString(index)+":"+line+`
__u32 Port_`+name+` = __builtin_bswap16(packet.InPort);
__u8 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &Port_`+name+`);
if (IsFound_`+name+` && *IsFound_`+name+`==1) {
`+RecodeHelper("0",path)+`
	return XDP_DROP;
}
`
				}


				if MAPS[name] == "CIDRv4" {
					rules += `
//# `+ToString(index)+":"+line+`
if (packet.Family == 2) {
	struct IPv4CIDR Key_`+name+` = { .prefixlen = 32, .addr      = packet.IP.Addr.IPv4 };
	__u8 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &Key_`+name+`);
	if (IsFound_`+name+`) {
`+RecodeHelper("0",path)+`
		return XDP_DROP;
	}
}
`
				}

				if MAPS[name] == "CIDRv6" {
					rules += `
//# `+ToString(index)+":"+line+`
if (packet.Family == 10) {
	struct IPv6CIDR Key_bots_v6 = {
    .prefixlen = 32,
    .addr = {
        .IPv6 = {
            packet.IP.Addr.IPv6[0],
            packet.IP.Addr.IPv6[1]
        }
    }
	};
	//struct IPv6CIDR Key_`+name+` = { .prefixlen = 128, .addr      = packet.IP.Addr.IPv6 };
	__u8 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &Key_`+name+`);
	if (IsFound_`+name+`) {
`+RecodeHelper("0",path)+`
		return XDP_DROP;
	}
}
`
				}

			// DROP;
			}else if(len(words)==1){
				rules += "//# "+ToString(index)+":"+line+"\n"
				rules += RecodeHelper("0",path)
				rules += "\t\treturn XDP_DROP;\n"
				hasReturn = 1
			}
		}



		// NEXT 
		if words[0]=="NEXT"{
			// NEXT black;
			if len(words)== 2{
				name := words[1]
				/*
				if MAPS[name] == "IP_DURATION"{
					rules += `
//# `+ToString(index)+":"+line+`
__u64 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &packet.IP);
if (IsFound_`+name+`) {
	__u64 now = bpf_ktime_get_ns();
	if (*IsFound_`+name+` > now){
`+RecodeHelper("0",path)+`
		return XDP_DROP;
	}else{
		bpf_map_delete_elem(&`+name+`, &packet.IP);
	}
}
`
				}
*/

				if MAPS[name] == "IP"{
					rules += `
//# `+ToString(index)+":"+line+`
__u64 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &packet.IP);
if (IsFound_`+name+`) {}else{
`+RecodeHelper("0",path)+`
	return XDP_DROP;
}
`
				}
				
				if MAPS[name] == "OTPORT"{
					rules += `
//# `+ToString(index)+":"+line+`
__u32 Port_`+name+` = __builtin_bswap16(packet.OtPort);
__u8 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &Port_`+name+`);
if (IsFound_`+name+` && *IsFound_`+name+`==1) {}else{
`+RecodeHelper("0",path)+`
	return XDP_DROP;
}
`
				}

				if MAPS[name] == "INPORT"{
					rules += `
//# `+ToString(index)+":"+line+`
__u32 Port_`+name+` = __builtin_bswap16(packet.InPort);
__u8 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &Port_`+name+`);
if (IsFound_`+name+` && *IsFound_`+name+`==1) {}else{
`+RecodeHelper("0",path)+`
	return XDP_DROP;
}
`
				}

			
				if MAPS[name] == "CIDRv4" {
					rules += `
//# `+ToString(index)+":"+line+`
if (packet.Family == 2) {
	struct IPv4CIDR Key_`+name+` = { .prefixlen = 32, .addr      = packet.IP.Addr.IPv4 };
	__u8 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &Key_`+name+`);
	if (IsFound_`+name+`) {}else{
`+RecodeHelper("0",path)+`
		return XDP_DROP;
	}
}
`
				}

				if MAPS[name] == "CIDRv6" {
					rules += `
//# `+ToString(index)+":"+line+`
if (packet.Family == 10) {
	struct IPv6CIDR Key_bots_v6 = {
    .prefixlen = 32,
    .addr = {
        .IPv6 = {
            packet.IP.Addr.IPv6[0],
            packet.IP.Addr.IPv6[1]
        }
    }
	};
	//struct IPv6CIDR Key_`+name+` = { .prefixlen = 128, .addr      = packet.IP.Addr.IPv6 };
	__u8 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &Key_`+name+`);
	if (IsFound_`+name+`) {}else{
`+RecodeHelper("0",path)+`
		return XDP_DROP;
	}
}
`
				}
			
		}
	}





		if words[0]=="PASS"{
			// PASS white;
			if len(words)== 2{
				name := words[1]
				if MAPS[name] == "IP_DURATION"{
					rules += `
//# `+ToString(index)+":"+line+`
__u64 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &packet.IP);
if (IsFound_`+name+`) {
	__u64 now = bpf_ktime_get_ns();
	if (*IsFound_`+name+` > now){
`+RecodeHelper("1",path)+` 
		return XDP_PASS;
	}else{
		bpf_map_delete_elem(&`+name+`, &packet.IP);
	}
}
`

				}

				if MAPS[name] == "IP"{
					rules += `
//# `+ToString(index)+":"+line+`
__u64 *IsFound_`+name+` = bpf_map_lookup_elem(&`+name+`, &packet.IP);
if (IsFound_`+name+`) {
`+RecodeHelper("1",path)+` 
	return XDP_PASS;
}
`
				}

			// PASS;
			}else if(len(words)==1){
				rules += "//# "+ToString(index)+":"+line+"\n"
				rules += RecodeHelper("1", path)
				rules += "\t\treturn XDP_PASS;\n"
				hasReturn = 1
			}
		}

		if words[0]=="GUARD" && words[1]=="SYNFloodAttack"{
			rules += GuardSYNFloodAttack(index,line,words)
		}
		if words[0]=="GUARD" && words[1]=="FragmentAttack"{
			rules += GuardFragmentAttack(index,line,words)
		}


		if strings.HasPrefix(line,"IF ") {
			rules += "//# "+ToString(index)+":"+line+"\n"
			rules += "if( "+ strings.TrimPrefix(line,"IF ") +" ){\n"
		}
		if strings.HasPrefix(line,"ELSE IF "){
			rules += "//# "+ToString(index)+":"+line+"\n"
			rules += "}else if( "+ strings.TrimPrefix(line,"ELSE IF ") +" ){\n"
		}
		if line=="ELSE"{
			rules += "//# "+ToString(index)+":"+line+"\n"
			rules += "}else{\n"
		}
		if line=="END"{
			rules += "//# "+ToString(index)+":"+line+"\n"
			rules += "}\n"
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



func GuardFragmentAttack(index int,line string,words []string) string{

//! BURAYA GUARD Fragment "REASON" gelecek
return `
//# `+ToString(index)+":"+line+`
if (packet.IsFragmented) {
	Log(&packet, 0x46524147, 0); // "FRAG"	
	return XDP_DROP;
}
`

}


func GuardSYNFloodAttack(index int,line string,words []string) string{
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
//# `+ToString(index)+":"+line+`
if (packet.Protocol == IPPROTO_TCP) {
		if (!CheckTCPFlags(&packet.TCP)) {
				//log_tcp_anomaly_drop(packet.Length);
				//send_log(0, &packet); // Test için anında logla
				return XDP_DROP; // Geçersiz TCP Bayrak Kombinasyonu (Nmap Stealth Scan vs.)
		}

		// SYN Flood Hız Sınırı (Rate Limit)
		if (packet.TCP.Flags & 0x02) { // 0x02 == SYN
				__u32 syn_key = 0;
				__u64 *global_syn = bpf_map_lookup_elem(&SYN_STATS, &syn_key);
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
								if (syn_val->count > 50) { // Saniyede 30 SYN'den fazlaysa
										__u64 guard_syn_flood_attack_time_out= now + `+expireNSStr+`;
										bpf_map_update_elem(&`+name+`, &packet.IP, &guard_syn_flood_attack_time_out, BPF_ANY);
										
										//log_syn_flood_drop(packet.Length);
										//send_log(0, &packet); // Test için anında logla
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


type IPv4CIDR struct {
	Prefixlen uint32
	Addr      uint32
}


func GetIPv4CIDRKey(value string) (IPv4CIDR, error) {
	ip, network, err := net.ParseCIDR(value)
	if err != nil {
		return IPv4CIDR{}, err
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return IPv4CIDR{}, fmt.Errorf("not an IPv4 CIDR: %s", value)
	}

	networkIP := network.IP.To4()
	if networkIP == nil {
		return IPv4CIDR{}, fmt.Errorf("not an IPv4 network: %s", value)
	}

	ones, bits := network.Mask.Size()
	if bits != 32 {
		return IPv4CIDR{}, fmt.Errorf("not an IPv4 network: %s", value)
	}


	cidr := IPv4CIDR{
		Prefixlen: uint32(ones),
		Addr:      binary.LittleEndian.Uint32(networkIP),
	} 

	write(cidr)
	return cidr , nil


}


type IPv6CIDR struct {
	Prefixlen uint32
	_         uint32
	Addr      [2]uint64
}

func GetIPv6CIDRKey(value string) (IPv6CIDR, error) {
	ip, network, err := net.ParseCIDR(value)
	if err != nil {
		return IPv6CIDR{}, err
	}

	ip6 := ip.To16()
	if ip6 == nil || ip.To4() != nil {
		return IPv6CIDR{}, fmt.Errorf("not an IPv6 CIDR: %s", value)
	}

	networkIP := network.IP.To16()
	if networkIP == nil || networkIP.To4() != nil {
		return IPv6CIDR{}, fmt.Errorf("not an IPv6 network: %s", value)
	}

	ones, bits := network.Mask.Size()
	if bits != 128 {
		return IPv6CIDR{}, fmt.Errorf("not an IPv6 network: %s", value)
	}

	cidr := IPv6CIDR{
		Prefixlen: uint32(ones),
		Addr: [2]uint64{
			binary.LittleEndian.Uint64(networkIP[0:8]),
			binary.LittleEndian.Uint64(networkIP[8:16]),
		},
	}

	write(cidr)
	return cidr, nil
}



func KtimeNS() uint64 {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return 0
	}
	return uint64(ts.Sec)*1e9 + uint64(ts.Nsec)
}
func Add(name string, value string, duration uint64, debug bool){

	file   := "/sys/fs/bpf/map_"+name
	m, err := ebpf.LoadPinnedMap(file, nil)
	if err != nil {
		write("[x] load pinned map: %s %v", file, err)
	}
	defer m.Close()

	

	if(MAPS[name]=="IP_DURATION"){
		key, err := GetIPKey(value)
		if err != nil {
			if(debug){
				write("[X] Ip not readable", value)
			}
			return 
		}
		var val uint64 = KtimeNS() + duration*1e9
		err = m.Update(key, val, ebpf.UpdateAny)
		if err != nil {
			write(err)
		}
		if(debug){
			write ("[+] success: Add(",name,value,duration,")")
		}
		return
	}else if( MAPS[name]=="IP" ){
		key, err := GetIPKey(value)
		if err != nil {
			if(debug){
				write("[X] Ip not readable", value)
			}
			return 
		}
		var val uint8 = 1
		err = m.Update(key, val, ebpf.UpdateNoExist)
		if err != nil {
			write(err)
		}
		if(debug){
			write ("[+] success: Add(",name,value,duration,")")
		}
		return
	}else if MAPS[name] == "CIDRv4" {
		key, err := GetIPv4CIDRKey(value)
		if err != nil {
			write("[X] CIDRv4 not readable:", value, err)
			return
		}

		var val uint8 = 1
		err = m.Update(key, val, ebpf.UpdateNoExist)
		if err != nil {
			write("[X] map update failed:", err)
			return
		}

		if(debug){
			write ("[+] success: Add(",name,value,duration,")")
		}
		return
	}else if MAPS[name] == "CIDRv6" {
		key, err := GetIPv6CIDRKey(value)
		if err != nil {
			write("[X] CIDRv6 not readable:", value, err)
			return
		}

		var val uint8 = 1
		err = m.Update(key, val, ebpf.UpdateNoExist)
		if err != nil {
			write("[X] map update failed:", err)
			return
		}

		if(debug){
			write ("[+] success: Add(",name,value,duration,")")
		}
		return
	}else if MAPS[name]=="INPORT" || MAPS[name]=="OTPORT" {
		port := uint32(ToNumber(value))
		var val uint8 = 1
		err = m.Update(&port, &val, ebpf.UpdateAny)
    if err != nil {
      write("[X] port map update failed:", err)
      return
    }

		if(debug){
			write ("[+] success: Add(",name,value,duration,")")
		}
		return
	}else{
		write("[X] MAP Type not found",name,MAPS )
	}
	

	// ebpf.UpdateAny      // varsa güncelle, yoksa ekle
	// ebpf.UpdateExist    // sadece mevcut key'i güncelle
	// ebpf.UpdateNoExist  // sadece yoksa ekle

}




func Clear(name string) {
	file := "/sys/fs/bpf/map_" + name

	m, err := ebpf.LoadPinnedMap(file, nil)
	if err != nil {
		fmt.Printf("[x] load pinned map: %s %v\n", file, err)
		return
	}
	defer m.Close()

	info, err := m.Info()
	if err != nil {
		fmt.Printf("[x] map info: %v\n", err)
		return
	}

	keySize := info.KeySize

	key := make([]byte, keySize)
	nextKey := make([]byte, keySize)

	for {
		err := m.NextKey(key, nextKey)
		if err != nil {
			break
		}

		if err := m.Delete(nextKey); err != nil {
			fmt.Printf("[x] delete key: %v\n", err)
			break
		}

		// Bir sonraki aramada mevcut key olarak
		// silinen key'i kullan.
		copy(key, nextKey)
	}

	fmt.Printf("[+] cleared %s\n", file)
}




func Count(name string) {
	file := "/sys/fs/bpf/map_" + name

	m, err := ebpf.LoadPinnedMap(file, nil)
	if err != nil {
		fmt.Printf("[x] load pinned map: %s %v\n", file, err)
		return
	}
	defer m.Close()

	info, err := m.Info()
	if err != nil {
		fmt.Printf("[x] map info: %v\n", err)
		return
	}

	key := make([]byte, info.KeySize)
	nextKey := make([]byte, info.KeySize)

	count := 0

	for {
		err := m.NextKey(key, nextKey)
		if err != nil {
			break
		}

		count++

		copy(key, nextKey)
	}

	fmt.Printf("[+] %s count: %d\n", name, count)
}

func SetMode(code string){
	var client *UnixSocketClient = UnixSocketClientInit("/web/sockets/erlik.sock")
	response := client.Request("MODE,"+code)
	write(response)
	client.Close()
}
























func IFACE() string {
	out, _ := exec.Command("sh", "-c", "ip route show default | awk '{print $5}'").Output()
	return strings.TrimSpace(string(out))
}


func Cleanup() {
	os.Remove("/sys/fs/bpf/bozkurt_program")
	if l, err := link.LoadPinnedLink("/sys/fs/bpf/bozkurt_link", nil); err == nil {
		l.Close()
	}
	os.Remove("/sys/fs/bpf/bozkurt_link")
}

func Pin() {

	Cleanup()

	InitSystemD()
	ErlikRunning := CommandIsServiceRunning("erlik")

	ErlikRunning = false
	if( ErlikRunning ){
		CommandStopService("erlik")
	}


	write("[.] IFACE:",IFACE())
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

	// Mevcut pinned map'leri bul
	replacements := make(map[string]*ebpf.Map)

	for name := range spec.Maps {
		if name == ".rodata" || name==".bss" {
			continue
		}

		path := "/sys/fs/bpf/map_" + name

		// CLEAR varsa eski map'i sil, reuse etme
		if val, ok := CLEARS[name]; ok && val == 1 {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				write("[X] map remove:", name, err)
				return
			}

			continue
		}


		// Pin yoksa yeni map oluşturulacak
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		
		// Pinned map varsa yeni programa bunu kullandır
		oldMap, err := ebpf.LoadPinnedMap(path, nil)
		if err == nil {
			replacements[name] = oldMap
			write("[=] map will be reused:", path)
		} else if !os.IsNotExist(err) {
			write("[X] map load:", name, err)
			return
		}
	}

	// BPF objelerini oluştur
	coll, err := ebpf.NewCollectionWithOptions(spec, ebpf.CollectionOptions{
		MapReplacements: replacements,
	})
	if err != nil {
		write("[X] NewCollection:", err)
		return
	}




	// Map pinleme
	fmt.Println("\n=== Pin Maps ===")

	for name, m := range coll.Maps {
		if name == ".rodata" || name == ".bss" {
			continue
		}

		path := "/sys/fs/bpf/map_" + name

		// CLEAR yapılmadıysa ve map zaten varsa:
		// MapReplacements sayesinde bu zaten eski map.
		// Tekrar Pin() yapmaya gerek yok.
		if _, err := os.Stat(path); err == nil {
			write("[=] map reused:", path)
		} else if os.IsNotExist(err) {
			// Yeni oluşturulan map'i pinle
			if err := m.Pin(path); err != nil {
				write("[X] map pin:", name, err)
				continue
			}

			write("[+] map pinned:", path)
		} else {
			write("[X] map stat:", name, err)
			continue
		}

		// ADD HER ZAMAN çalışacak
		if list, ok := ADDS[name]; ok {
			for _, item := range list {
				Add(name, item, 0, true)
			}
		}
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

	
	if( ErlikRunning ){
		CommandStartService("erlik")
	}

}





















func CommandStartService(service string) {
	var job dbus.ObjectPath

	err := Systemd.Call(
		"org.freedesktop.systemd1.Manager.StartUnit",
		0,
		service+".service",
		"replace",
	).Store(&job)

	if err != nil {
		write("[X] "+service+" service not started:", err)
	} else {
		write("[+] "+service+" service started")
	}
}


func CommandStopService(service string) {
	var job dbus.ObjectPath

	err := Systemd.Call(
		"org.freedesktop.systemd1.Manager.StopUnit",
		0,
		service+".service",
		"replace",
	).Store(&job)

	if err != nil {
		write("[X] "+service+" service not stopped:", err)
	} else {
		write("[+] "+service+" service stopped")
	}
}


func CommandRestartService(service string) {
	var job dbus.ObjectPath

	err := Systemd.Call(
		"org.freedesktop.systemd1.Manager.RestartUnit",
		0,
		service+".service",
		"replace",
	).Store(&job)

	if err != nil {
		write("[X] "+service+" service not restarted:", err)
	} else {
		write("[+] "+service+" service restarted")
	}
}


func CommandReloadService(service string) {
	var job dbus.ObjectPath

	err := Systemd.Call(
		"org.freedesktop.systemd1.Manager.ReloadUnit",
		0,
		service+".service",
		"replace",
	).Store(&job)

	if err != nil {
		write("[X] "+service+" service not reloaded:", err)
	} else {
		write("[+] "+service+" service reloaded")
	}
}

func CommandEnableService(service string) {

	err := Systemd.Call(
		"org.freedesktop.systemd1.Manager.EnableUnitFiles",
		0,
		[]string{service + ".service"},
		false,
		true,
	).Err

	if err != nil {
		write("[X] "+service+" service not enabled:", err)
	} else {
		write("[+] "+service+" service enabled")
	}
}

func CommandDisableService(service string) {

	err := Systemd.Call(
		"org.freedesktop.systemd1.Manager.DisableUnitFiles",
		0,
		[]string{service + ".service"},
		false,
	).Err

	if err != nil {
		write("[X] "+service+" service not disabled:", err)
	} else {
		write("[+] "+service+" service disabled")
	}
}


func CommandDaemonReload() {

	err := Systemd.Call(
		"org.freedesktop.systemd1.Manager.Reload",
		0,
	).Err

	if err != nil {
		write("[X] systemd daemon reload failed:", err)
	} else {
		write("[+] systemd daemon reload completed")
	}
}



func CommandIsServiceRunning(service string) bool {
	
	var units []struct {
		Name        string
		Description string
		LoadState    string
		ActiveState string
		SubState    string
		Follow      string
		Path        dbus.ObjectPath
		JobId       uint32
		JobType     string
		JobPath     dbus.ObjectPath
	}

	call := Systemd.Call(
		"org.freedesktop.systemd1.Manager.ListUnits",
		0,
	)

	if call == nil {
		return false
	}

	if err := call.Store(&units); err != nil {
		return false
	}

	if !strings.HasSuffix(service, ".service") {
		service += ".service"
	}

	for _, unit := range units {
		if unit.Name == service {
			return unit.ActiveState == "active"
		}
	}

	return false

}
