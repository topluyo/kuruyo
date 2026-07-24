package main

import (
	"os"
	"os/exec"
	"strings"
	"fmt"
	"github.com/godbus/dbus/v5"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"
)



const SYSTEMD_PATH = "/etc/systemd/system/"
var Shell *BashShell
var Systemd  dbus.BusObject
func main() {
	Shell = NewBashShell()
	defer Shell.Close()



	conn, err := dbus.SystemBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	Systemd = conn.Object("org.freedesktop.systemd1","/org/freedesktop/systemd1")

	if len(os.Args) != 0{
		command(strings.Join(os.Args," "))
	}else{
		KurucuInfo()
		UnixSocketServer("/web/sockets/KURUCU.sock", func(request string) string {
			return command(request)
		})
	}
}


func command(cmd string) string{
	//- create=server:port
	create_argument := arguments(cmd, "create")
	delete_argument := arguments(cmd, "delete")
	  port_argument := arguments(cmd, "port")
		kill_argument := arguments(cmd, "kill")
	reload_argument := arguments(cmd, "reload")
	   log_argument := arguments(cmd, "log")
		 run_argument := arguments(cmd, "run")
		info_argument := arguments(cmd, "info")
 process_argument := arguments(cmd, "process")
	
	if( create_argument != "" ){
		server_ports := strings.Split(create_argument,":")
		if len(server_ports)==1 && ToNumber(server_ports[0])!=0 {
			// start=15
		}
		if len(server_ports)==1 && ToNumber(server_ports[0])==0 {
			// start=system
			server := server_ports[0]
			config := Load("/web/config/"+server+"/routers.json")
			CreateService(config, server)
		}
		if len(server_ports)==2 && ToNumber(server_ports[0])==0 &&  ToNumber(server_ports[1])!=0 {
			// start=system:1924
			server := server_ports[0]
			port   := ToNumber(server_ports[1])
			config := Load("/web/config/"+server+"/routers.json")
			CreateServicePort(config, server, port)
		}

		return ""
	}else if delete_argument!=""{

		server_ports := strings.Split(delete_argument,":")
		if len(server_ports)==1 && ToNumber(server_ports[0])!=0 {
			// delete=15
			port   := ToNumber(server_ports[0])
			DeleteWithPort(port)
		}
		if len(server_ports)==1 && ToNumber(server_ports[0])==0 {
			// delete=system
			server := server_ports[0]
			config := Load("/web/config/"+server+"/routers.json")
			DeleteWithPort(config.HTTP)
			//DeleteWithName(server_ports[0])
		}
		if len(server_ports)==2 && ToNumber(server_ports[0])==0 &&  ToNumber(server_ports[1])!=0 {
			// delete=system:1924
		}

		
		
		return ""
	}else if reload_argument!=""{

		server_ports := strings.Split(reload_argument,":")
		if len(server_ports)==1 && ToNumber(server_ports[0])!=0 {
			// delete=15
			port   := ToNumber(server_ports[0])
			Reload(port)
		}
		if len(server_ports)==1 && ToNumber(server_ports[0])==0 {
			// delete=system
			server := server_ports[0]
			config := Load("/web/config/"+server+"/routers.json")
			Reload(config.HTTP)
		}
		if len(server_ports)==2 && ToNumber(server_ports[0])==0 &&  ToNumber(server_ports[1])!=0 {
			// delete=system:1924
		}

		
		
		return ""
	}else if port_argument!=""{

		server_ports := strings.Split(port_argument,":")
		if len(server_ports)==1 && ToNumber(server_ports[0])!=0 {
			// port=15
			port   := ToNumber(server_ports[0])
			Port(port)
		}
		
		
		return ""
	}else if kill_argument!=""{

		server_ports := strings.Split(kill_argument,":")
		if len(server_ports)==1 && ToNumber(server_ports[0])!=0 {
			// port=15
			port   := ToNumber(server_ports[0])
			Kill(port)
		}
		
		
		return ""
	}else if arguments(cmd,"check")!=""{

		lang_path := strings.Split(arguments(cmd,"check"),":")
		if len(lang_path)==2 {
			lang   := lang_path[0]
			path   := lang_path[1]
			if(lang=="go"){
				Check_Go(path,"")
			}
		}
		
		return ""
	}else if log_argument!=""{

		
		server_ports := strings.Split(log_argument,":")
		if len(server_ports)==1 && ToNumber(server_ports[0])!=0 {
			// log=15
			port   := ToNumber(server_ports[0])
			LogTail(port)
		}
		if len(server_ports)==1 && ToNumber(server_ports[0])==0 {
			// log=system
			server := server_ports[0]
			config := Load("/web/config/"+server+"/routers.json")
			LogTail(config.HTTP)
		}

		
		return ""
	}else if info_argument!=""{

		
		server_ports := strings.Split(info_argument,":")
		if len(server_ports)==1 && ToNumber(server_ports[0])!=0 {
			// log=15
			port   := ToNumber(server_ports[0])
			Port(port)
		}
		if len(server_ports)==1 && ToNumber(server_ports[0])==0 {
			// log=system
			server := server_ports[0]
			config := Load("/web/config/"+server+"/routers.json")
			Info(config)
		}

		
		return ""
	}else if process_argument!=""{

		pid := ToNumber(process_argument)
		ProcessInfo(pid)
		
		return ""
	}else if run_argument!=""{

		pid := ToNumber(run_argument)
		RunPort(pid)
		
		return ""
	}else{

	}

	

	KurucuInfo()
	return ""
}

func KurucuInfo(){
	table("KURUCU STARTED")
	write("    create=service service:port")
	write("    delete=service|port")
	write("      info=service|port")
	write("")
	write("   process=pid")
	write("      port=port")
	write("      kill=port")
	write("    reload=port")
	write("       log=port")


	write("     check=lang:path")

	write("       run=port  RUN this program")
	//write("     start=service:port")
	//write("      stop=service:port")
	
}


func Run(command string) error {
	cmd := exec.Command("bash", "-c", command)
	return cmd.Run()
}

func Bash(command string) (string, error) {
	cmd := exec.Command("bash","-c",command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func BashStream(command string) error {
	cmd := exec.Command("bash","-c",command)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	cmd.Start()
	go func(){
		buf := make([]byte,1024)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				return
			}
			fmt.Print(string(buf[:n]))
		}
	}()
	go func(){
		buf := make([]byte,1024)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				return
			}
			fmt.Print(string(buf[:n]))
		}
	}()
	return cmd.Wait()
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













func FindServiceWithPortName(port int) string {
	serviceDir := SYSTEMD_PATH

	files, err := os.ReadDir(serviceDir)
	if err != nil {
		write("[X] -"+ToString(port)+".service not found")
		return ""
	}

	portRegex := regexp.MustCompile(`(?m)(?:port|PORT|--port|-p)[=\s:]?` + strconv.Itoa(port) + `\b`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ".service" {
			continue
		}

		path := filepath.Join(serviceDir, file.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		if portRegex.Match(data) {
			name := strings.TrimSuffix(file.Name(),".service") 
			write("[+] SERVICE="+name+" found with name=*-"+ToString(port)+".service")
			return name
		}
	}

	write("[X] -"+ToString(port)+".service not found")
	return ""
}

/*
func FindPIDFromNetPort(port int, hide ...bool) int {
	silent := false
	if len(hide) > 0 { silent = hide[0] }

	cmd := exec.Command("sh", "-c",
		fmt.Sprintf(`ss -tulnp | awk '/:%d /&&/pid=/{match($0,/pid=([0-9]+)/,m);print m[1]}'`, port),
	)

	output, err := cmd.Output()
	if err != nil {
		if !silent { write("[X] ss command failed:", err) }
		return 0
	}

	pidStr := strings.TrimSpace(string(output))

	if pidStr == "" {
		if !silent { write("[X] PID not found for port:", port) }
		return 0
	}

	
	pidStr = strings.TrimSpace(pidStr)
	lines := strings.Split(pidStr, "\n")
	pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		if !silent { write("[X] PID parse failed:", err, pidStr) }
		return 0
	}

	if !silent { write("[+] PID="+ToString(pid)+" found from port="+ToString(port)) }
	return pid
}
*/



/*
func FindPIDFromNetPort(port int, hide ...bool) int {
	silent := len(hide) > 0 && hide[0]

	portHex := fmt.Sprintf("%04X", port)

	// TCP + TCP6 kontrolü
	files := []string{
		"/proc/net/tcp",
		"/proc/net/tcp6",
	}

	var inode string

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")

		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) < 10 {
				continue
			}

			local := fields[1]
			parts := strings.Split(local, ":")

			if len(parts) != 2 {
				continue
			}

			if parts[1] == portHex {
				inode = fields[9]
				break
			}
		}

		if inode != "" {
			break
		}
	}

	if inode == "" {
		if !silent {
			write("[X] PID not found for port:", port)
		}
		return 0
	}

	// inode -> PID eşleştirme
	
	if err != nil {
		return 0
	}

	target := "socket:[" + inode + "]"

	for _, fd := range entries {
		link, err := os.Readlink(fd)
		if err != nil {
			continue
		}

		if link == target {
			pidStr := strings.Split(fd, "/")[2]

			pid, err := strconv.Atoi(pidStr)
			if err == nil {
				if !silent {
					write("[+] PID="+ToString(pid)+" found from port="+ToString(port))
				}
				return pid
			}
		}
	}

	if !silent {
		write("[X] PID not found")
	}

	return 0
}

*/




var (
	pidCacheMu sync.Mutex
	pidCache   = map[int]int{}
	pidCacheAt time.Time
)

func refreshPIDCache() {
	cmd := exec.Command("sh", "-c",
		`ss -tulnp | awk '/pid=/{match($0,/:[0-9]+/,p); match($0,/pid=[0-9]+/,i); if(p[0]!="" && i[0]!="") {gsub(":","",p[0]); gsub("pid=","",i[0]); print p[0]" "i[0]}}'`,
	)

	output, err := cmd.Output()
	if err != nil {
		return
	}


	
	newCache := make(map[int]int)

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		port, err1 := strconv.Atoi(fields[0])
		pid, err2 := strconv.Atoi(fields[1])

		if err1 == nil && err2 == nil {
			newCache[port] = pid
		}
	}

	pidCache = newCache
	pidCacheAt = time.Now()
}

func FindPIDFromNetPort(port int, hide ...bool) int {
	silent := false
	if len(hide) > 0 {
		silent = hide[0]
	}

	pidCacheMu.Lock()

	// 1 saniyelik cache kontrolü
	if time.Since(pidCacheAt) > time.Second || pidCacheAt.IsZero() {
		refreshPIDCache()
	}

	pid := pidCache[port]

	pidCacheMu.Unlock()

	if pid == 0 {
		if !silent {
			write("[X] PID not found for port:", port)
		}
		return 0
	}

	if !silent {
		write("[+] PID=" + ToString(pid) + " found from port=" + ToString(port))
	}

	return pid
}

func FindServiceNameFromPID(pid int) string {
	path := fmt.Sprintf("/proc/%d/cgroup", pid)

	data, err := os.ReadFile(path)
	if err != nil {
		write("[X] pid cgroup not found:", pid)
		return ""
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.Contains(line, ".service") {
			parts := strings.Split(line, "/")

			for _, part := range parts {
				if strings.HasSuffix(part, ".service") {
					name := strings.TrimSuffix(part, ".service")
					write("[+] SERVICE="+name+" found with PID="+ToString(pid))
					return name
				}
			}
		}
	}

	write("[X] service not found for pid:", pid)
	return ""
}





func GetMemoryRSS(pid int) (int64) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseInt(fields[1], 10, 64)
				if err != nil {
					return 0
				}
				return kb * 1024 // byte
			}
		}
	}

	return 0
}



func GetProcessInfo(pid int) {
	cmdline, _ := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	command := strings.ReplaceAll(string(cmdline), "\x00", " ")

	cwd, _ := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	exe, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))

	write(fmt.Sprintf(" ┌─ PROCESS PID:"), strconv.Itoa(pid))
	write(" │  CMD:", command)
	write(" │  PWD:", cwd)
	write(" │  EXE:", exe)

	parents := getParentChain(pid)

	for i, parent := range parents {
		isLast := "│ "
		if(len(parents)==i+1){
			isLast = "└─"
		}
		write(fmt.Sprintf(" ├─ PARENT[%d] PID:", i+1), strconv.Itoa(parent.PID))
		write(fmt.Sprintf(" │  CMD:"), parent.Cmd)
		write(fmt.Sprintf(" │  PWD:"), parent.Cwd)
		write(fmt.Sprintf(" "+isLast+" EXE:"), parent.Exe)
	}
	
}

type ProcessParent struct {
	PID int
	Cmd string
	Cwd string
	Exe string
}

func getParentChain(pid int) []ProcessParent {
	var parents []ProcessParent
	visited := make(map[int]bool)

	currentPID := pid

	for {
		ppid := getParentPID(currentPID)

		if ppid <= 0 || visited[ppid] {
			break
		}

		visited[ppid] = true

		cmdline, _ := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", ppid))
		cmd := strings.TrimSpace(strings.ReplaceAll(string(cmdline), "\x00", " "))

		exe, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", ppid))

		cwd, _ := os.Readlink(fmt.Sprintf("/proc/%d/cwd", ppid))


		parents = append(parents, ProcessParent{
			PID: ppid,
			Cmd: cmd,
			Exe: exe,
			Cwd: cwd,
		})

		currentPID = ppid
	}

	return parents
}

func getParentPID(pid int) int {
	status, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return -1
	}

	for _, line := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(line, "PPid:") {
			fields := strings.Fields(line)
			if len(fields) == 2 {
				ppid, err := strconv.Atoi(fields[1])
				if err == nil {
					return ppid
				}
			}
		}
	}

	return -1
}


func WriteServiceFile(path string, file string) error {
	err := os.WriteFile(SYSTEMD_PATH + path+".service",[]byte(file),0644)
	if err != nil {
		write("[X] service file write failed:", err)
		return err
	}
	write("[+] service file created:", path)
	return nil
}
func DeleteServiceFile(name string) error {
	path := SYSTEMD_PATH + name + ".service"
	err := os.Remove(path)
	if err != nil {
		write("[X] service file delete failed:", err)
		return err
	}
	write("[+] service file deleted:", name)
	return nil
}

func LogServiceFile(name string) {
	path := SYSTEMD_PATH + name + ".service"

	content, err := os.ReadFile(path)
	if err != nil {
		write("[X] service file read failed:", err)
		return
	}

	write("[+] service file content:\n\n    " + strings.ReplaceAll(string(content), "\n", "\n    ") + "\n")
}