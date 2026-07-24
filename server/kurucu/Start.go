package main


import(
  "strings"
  "fmt"
  "unicode"
  "sort"
  "os"
)



func CreateService(conf *Config, server string){
  if(conf==nil){
    write("[X] configuration not found")
    return 
  }
  
  name := "kuruyo-"+Slugify(server)
  write("[.]", name, server, conf.HTTP, conf.HTTPS)


  app  := "go.app"
  path := "/web/server/"+conf.Engine
  lang := "go"

  if(lang=="go"){

    Check_Go(path,app)
    exec := path+"/"+app+" config=/web/config/"+server+"/routers.json"
    
    serviceFileContent := CreateServiceFileContent(server+" SERVER",exec,path)
    write(serviceFileContent)
    WriteServiceFile(name,serviceFileContent)
    CommandDaemonReload()
    CommandEnableService(name)
    CommandRestartService(name)
  }

}

func CreateServicePort(conf *Config, server string, port int){
  if(conf==nil){
    write("[X] configuration not found")
    return 
  }
  var route *Route = LoadRoute(conf,port)
  if(route==nil){
    write("[X] port not found in configuration file")
    return
  }

  lang_path := strings.Split(route.Serve,":")
  if len(lang_path)<2{
    write("[X] serve:'"+route.Serve+"' is not valid.")
    return
  }

  lang := lang_path[0]
  path := lang_path[1]
  baseList := strings.Split(route.Name, "/")
  base := "/" + strings.Join(baseList[1:], "/")
  
  name := "kuruyo-"+Slugify(server)+"-"+ToString(port)
  write("[.]", name, server, port, lang, path, base)

  app  := Slugify(route.Name)+".app"  
  app   = "go.app"

  
  if(lang=="go"){


    Check_Go(path,app)
    exec := path+"/"+app+" port="+ToString(port)+" base=\""+base+"\" path=\""+path+"\""

    serviceFileContent := CreateServiceFileContent(route.Description,exec,path)

    WriteServiceFile(name,serviceFileContent)
    CommandDaemonReload()
    CommandEnableService(name)
    CommandRestartService(name)

  }

}

func Check_Go(path string, app string){
  if err := Run("cd "+path+" && /program/go mod init app > /dev/null 2>&1") ; err!=nil && err.Error()!="exit status 1" {
    write("[X] go mod init build error",err)
  }else{
    write("[+] go mod init app")
  }
  if err := Run("cd "+path+" && /program/go mod tidy > /dev/null 2>&1") ; err!=nil {
    write("[X] go mod tidy build error",err)
  }else{
    write("[+] go mod tidy")
  }
  if err := Run("cd "+path+" && /program/go vet . > /dev/null 2>&1") ; err!=nil {
    write("[X] go vet error",err)
    write(Bash("cd "+path+" && /program/go vet ."))
  }else{
    write("[+] go vet")
  }
  if app!=""{
    if err := Run("cd "+path+" && /program/go build -o "+app+" . > /dev/null 2>&1") ; err!=nil {
      write("[X] go build error",err)
      write(Bash("cd "+path+" && /program/go build -o "+app+" ."))
    }else{
      write("[+] go build")
    }
  }
}



func DeleteWithPort(port int){
  service := ""
  service  = FindServiceWithPortName(port)
  if(service!=""){
    write("[.] deleting ", service)
    CommandStopService(service)
    CommandDisableService(service)
    DeleteServiceFile(service)
    CommandDaemonReload()
    return
  }
  pid := FindPIDFromNetPort(port)
  if(pid!=0){
    service = FindServiceNameFromPID(pid)
    if(service!=""){
      write("[.] deleting ", service)
      CommandStopService(service)
      CommandDisableService(service)
      DeleteServiceFile(service)
      CommandDaemonReload()
      return
    }
  }
}


func Port(port int){
  pid := FindPIDFromNetPort(port)
  FindServiceWithPortName(port)
  if(pid!=0){
    service := FindServiceNameFromPID(pid)
    if(service!=""){
      write("[+]",service)
      LogServiceFile(service)
    }
    GetProcessInfo(pid)
    write("[+] RAM:", ToString(int(GetMemoryRSS(pid)/1024/1024)) ,"MB" )
  }
}


func ProcessInfo(pid int){
  service := FindServiceNameFromPID(pid)
  if(service!=""){
    write("[+]",service)
    LogServiceFile(service)
  }
  GetProcessInfo(pid)
  write("[+] RAM:", ToString(int(GetMemoryRSS(pid)/1024/1024)) ,"MB" )
}

func Reload(port int){
  pid := FindPIDFromNetPort(port)
  if(pid!=0){
    service := FindServiceNameFromPID(pid)
    if(service!=""){
      CommandReloadService(service)
      GetProcessInfo( pid )
      write("[+] RAM:", ToString(int(GetMemoryRSS(pid)/1024/1024)) ,"MB" )
    }
  }
  
}



func Kill(port int){
  service := ""
  
  pid := FindPIDFromNetPort(port)
  
  if service = FindServiceWithPortName(port); service!=""{
    CommandStopService(service)
    CommandDisableService(service)
    DeleteServiceFile(service)
    CommandDaemonReload()
  }
  
  if pid!=0 {
    write("[.] Terminating PID="+ToString(pid))
    Run("kill "+ToString(pid))
  }
}



func LogTail(port int){
  pid := FindPIDFromNetPort(port)
  FindServiceWithPortName(port)
  if(pid!=0){
    service := FindServiceNameFromPID(pid)
    if(service!=""){
      write("[+]",service)
      LogServiceFile(service)
      GetProcessInfo(pid)
      write("[+] RAM:", ToString(int(GetMemoryRSS(pid)/1024/1024)) ,"MB" )
      BashStream("journalctl -n 40 -f -u "+service)
    }else{
      write("[X] PORT="+ToString(port)+"service not found")
    }
  }
}


type InfoSys struct{
  PID int
  RAM int
  PORT int 
  HOST string
}
func Info(c *Config){
  var infos []*InfoSys = make([]*InfoSys, 0)
  for h,r := range c.Routes{
    ports := Ranges(r.Ports)
    for _, p := range ports {
      port := ToNumber(p)
      infos = append(infos, InfoPort(port,h))
    } 
  }
  
  infos = append(infos, InfoPort(c.HTTP, "HTTP"))
  infos = append(infos, InfoPort(c.HTTPS, "HTTPS"))

  sort.Slice(infos, func(i, j int) bool {
		return infos[i].RAM < infos[j].RAM
	})

  for _, info := range infos {
		fmt.Printf(" %-7d %4dMB %6d  %-12s \n", info.PID, info.RAM, info.PORT, info.HOST)
	}
}

func InfoPort(port int, host string) *InfoSys{
  pid  := FindPIDFromNetPort( port, true )
  ram  := int(GetMemoryRSS(pid)/1024/1024)
  return &InfoSys{
    PID : pid,
    RAM : ram,
    PORT: port,
    HOST: host,
  }
  //fmt.Printf(" %-7d %4dMB %6d  %-12s \n", pid, ram, port, host)
}


func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func RunPort(port int){
  base := ""
  path := os.Getenv("PWD")
  write("[.]", path)
  if FileExists(path+"/main.go"){
    app := "run.app"
    Check_Go(path,app)
    exec := path+"/"+app+" port="+ToString(port)+" base=\""+base+"\" path=\""+path+"\""
    BashStream(exec)
  }else if FileExists(path+"/router.php") {
    exec := "/program/php -d upload_max_filesize=500M -d post_max_size=500M -S 0.0.0.0:"+ToString(port)+" -t "+path+" "+path+"/router.php";
    BashStream(exec)
  }else if FileExists(path+"/main.js") {
    
  }else{
    write("[X] main.go, router.php, main.js not found!")
  }
}



func CreateServiceFileContent(description, command, workdir string) string {

  //! BURADA MAX LimitNOFILE değeri bashden alınacak
  // cat /proc/sys/fs/file-max

	return fmt.Sprintf(`[Unit]
Description=%s 
After=network.target

[Service]
ExecReload=/bin/kill -SIGUSR1 $MAINPID
ExecStart=%s
Restart=always
User=%s
WorkingDirectory=%s
LimitNOFILE=4194304

[Install]
WantedBy=multi-user.target
`, description, command, "root", workdir)
}





func Slugify(input string) string {
	input = strings.ToLower(input)
	var result strings.Builder
	for _, r := range input {
		if r >= 'a' && r <= 'z' {
			result.WriteRune(r)
		} else if unicode.IsSpace(r) || r == '-' || r == '_' {
			continue
		}
	}
	return result.String()
}