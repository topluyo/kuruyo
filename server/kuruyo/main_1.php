<?php
error_reporting(E_ALL);
ini_set('display_errors', 1);


$SYSTEMD = "/etc/systemd/system/";
define("SYSTEMD","/etc/systemd/system/");


function slugify($text){
    $text = iconv('UTF-8', 'ASCII//TRANSLIT//IGNORE', $text);
    $text = preg_replace('/[^a-zA-Z0-9\-\.]+/', '_', $text);
    //$text = preg_replace('/_+/', '_', $text);
    $text = preg_replace('/[_-]+/', '-', $text);
    //$text = preg_replace('/\.+/', '.', $text);
    $text = trim($text, '_');
    $text = trim($text, '-');
    return $text;
}

function name($folder,$type="router"){
    //return "kuruyo-".preg_replace('/[^a-zA-Z0-9_-]/', '', $folder)."-".$type."-kuruyo";
    //@ NOW
    return preg_replace('/[^a-zA-Z0-9_-]/', '', $folder)."-".$type."-kuruyo";
}


function logc(string $msg, string $color = "frontWhite"){
    $colorMap = [
        'bold'       => '1;37',
        /*   Ön (foreground) renkler   */
        'frontRed'   => '1;31',
        'frontGreen' => '1;32',
        'frontBlue'  => '1;34',
        'frontAqua'  => '1;36',   // 3.6 cyan
        'frontYellow'=> '1;33',
        'frontWhite' => '37',
        'frontBlack' => '30',

        /*   Arka plan (background) renkler   */
        'backRed'    => '41',
        'backGreen'  => '42',
        'backBlue'   => '44',
        'backAqua'   => '46',   // 3.6 cyan
        'backYellow' => '43',
        'backWhite'  => '47',
        'backBlack'  => '40',

        /*   Özel kodlar   */
        'reset'      => '0',
    ];
    $code = $colorMap[$color] ?? $color;   // Eğer haritada yoksa doğrudan kullan
    echo "\033[" . $code . "m" . $msg . "\033[0m\n";
}


function json($data){
    echo json_encode($data,JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
}

function table($data,$limits=false){
    if(is_numeric(array_keys($data)[0])){
        if($limits){
            $_data=[];
            foreach($data as $row){
                $_row = [];
                foreach(explode(",",$limits) as $limit){
                    @$_row[$limit] = $row[$limit];
                }
                $_data[] = $_row;
            }
            $data = json_decode(json_encode($_data),true);
            table_view(array_keys($data[0]),$data);
            return;
        }
        table_view(array_keys($data[0]),$data);
    }else{
        if($limits){
            $_data=[];
            foreach(explode(",",$limits) as $limit){
                @$_data[$limit] = $data[$limit];
            }
            $data = json_decode(json_encode($_data),true);
            table_view(array_keys($data),[array_values($data)]);
        }else{
            table_view(array_keys($data),[array_values($data)]);
        }
    }
}

function table_view($header, $data) {



    $MAX = 40;

    // Metin kısaltma fonksiyonu
    $shorten = function($text) use ($MAX) {
        if(!$text) $text = "";
        if (mb_strlen($text) <= $MAX) return $text;
        $visible = $MAX - 4;
        return "...." . mb_substr($text, -$visible);
    };

    // Kolon genişlikleri
    $widths = [];
    foreach ($header as $i => $h) {
        $widths[$i] = min($MAX, mb_strlen($h));
    }

    foreach ($data as $row) {
        $i = 0;
        foreach ($row as $col) {
            $widths[$i] = @min($MAX, max($widths[$i], mb_strlen($col)));
            $i++;
        }
    }
    

    // Çizgi karakterleri
    $tl="┌"; $tr="┐"; $bl="└"; $br="┘";
    $h="─";  $v="│";
    $tm="┬"; $bm="┴"; $ml="├"; $mr="┤"; $mm="┼";

    // ÜST SINIR
    echo $tl;
    foreach ($widths as $i => $w) {
        echo str_repeat($h, $w + 2);
        echo ($i == array_key_last($widths)) ? $tr : $tm;
    }
    echo PHP_EOL;

    // BAŞLIK
    echo $v;
    foreach ($header as $i => $htext) {
        $txt = $shorten($htext);
        echo " " . str_pad($txt, $widths[$i]) . " " . $v;
    }
    echo PHP_EOL;

    // AYRAÇ
    echo $ml;
    foreach ($widths as $i => $w) {
        echo str_repeat($h, $w + 2);
        echo ($i == array_key_last($widths)) ? $mr : $mm;
    }
    echo PHP_EOL;

    // VERİ SATIRLARI
    foreach ($data as $row) {
        echo $v;
        $i = 0;
        foreach ($row as $col) {
            $txt = $shorten($col);
            echo " " . str_pad($txt, $widths[$i]) . " " . $v;
            $i++;
        }
        echo PHP_EOL;
    }

    // ALT SINIR
    echo $bl;
    foreach ($widths as $i => $w) {
        echo str_repeat($h, $w + 2);
        echo ($i == array_key_last($widths)) ? $br : $bm;
    }
    echo PHP_EOL;
}



function ranges(string $s): array {
    $s = trim($s);
    if (ctype_digit($s)) return [$s];

    if (str_contains($s, '-')) {
        [$a, $b] = explode('-', $s, 2);
    } elseif (str_contains($s, '+')) {
        [$a, $n] = explode('+', $s, 2);
        $b = (string)((int)$a + (int)$n);
    } else {
        throw new InvalidArgumentException("Invalid: $s");
    }

    if (!ctype_digit("$a") || !ctype_digit("$b") || (int)$a > (int)$b)
        throw new InvalidArgumentException("Invalid: $s");

    $out = [];
    for ($i = (int)$a; $i <= (int)$b; $i++) $out[] = (string)$i;
    return $out;
}


function bash($cmd)
{
    $descriptorspec = [
        0 => ["pipe", "r"],  // stdin
        1 => ["pipe", "w"],  // stdout
        2 => ["pipe", "w"],  // stderr
    ];

    // stdbuf ekleyerek buffer kapatıyoruz
    $process = proc_open("stdbuf -o0 -e0 " . $cmd, $descriptorspec, $pipes);

    if (!is_resource($process)) {
        return false;
    }

    // Non-blocking mod
    stream_set_blocking($pipes[1], false);
    stream_set_blocking($pipes[2], false);

    while (true) {
        $out = fgets($pipes[1]);
        $err = fgets($pipes[2]);

        if ($out !== false) {
            echo $out;
            ob_flush();
            flush();
        }

        if ($err !== false) {
            echo "[ERR] " . $err;
            ob_flush();
            flush();
        }

        // CPU %100 yemesin
        usleep(10000);
    }

    proc_close($process);
}



class Service{


    public static function bash(string $cmd, array $env = [], string $cwd = null): int{
        passthru($cmd);          // streams stdout and stderr directly

        return 0;

        if (!function_exists('posix_getuid') || posix_getuid() !== 0) {
            // Optionally prepend 'sudo -n' if you want to elevate
            // $cmd = 'sudo -n ' . $cmd;
        }

        // Set up descriptors: we want both stdout and stderr piped.
        $descriptors = [
            0 => ['pipe', 'r'],   // STDIN (unused)
            1 => ['pipe', 'w'],   // STDOUT
            2 => ['pipe', 'w'],   // STDERR
        ];

        $process = proc_open($cmd, $descriptors, $pipes, $cwd, $env);

        if (!is_resource($process)) {
            throw new RuntimeException("Unable to start command: $cmd");
        }

        // We close the STDIN pipe because we are not feeding input.
        fclose($pipes[0]);

        // Enable output buffering to avoid client buffering
        if (function_exists('apache_setenv')) {
            apache_setenv('no-gzip', 1);
        }
        header('Content-Type: text/plain');
        // Flush headers and any buffers that may be sent earlier
        flush();

        // Read both stdout and stderr concurrently
        $pipes[1] = fopen('php://temp', 'r+'); // for reading stdout
        $pipes[2] = fopen('php://temp', 'r+'); // for reading stderr

        // Reassign to the actual pipes for reading
        $pipes[1] = $pipes[1];
        $pipes[2] = $pipes[2];

        // Use a non-blocking read loop
        $stdout = $pipes[1];
        $stderr = $pipes[2];

        // Close the pipes after we are done
        while (true) {
            // Read a chunk from stdout
            $outChunk = fgets($stdout, 1024);
            if ($outChunk !== false) {
                echo $outChunk;
                flush();
            }

            // Read a chunk from stderr
            $errChunk = fgets($stderr, 1024);
            if ($errChunk !== false) {
                // Send stderr to same output or separate stream
                echo $errChunk;
                flush();
            }

            // Break if both pipes have reached EOF
            if ($outChunk === false && $errChunk === false) {
                $status = proc_get_status($process);
                if ($status['running'] === false) {
                    break;
                }
            }

            // Small sleep to avoid tight loop when there's no data
            usleep(20000); // 20 ms
        }

        // Close the pipes and the process
        fclose($stdout);
        fclose($stderr);
        return proc_close($process);
    }

    

    public static function findServiceInList($list){
        foreach($list as $item){
            if( substr($item, -8) === '.service' ){
                return $item;
            }
        }
    }

    public static function ActivePorts($starts){
        global $SYSTEMD;
        $files = glob($SYSTEMD . "$starts*.service");
        $response = [];
        foreach ($files as $file) {
            if (is_file($file)) {
                $response[] = $file;
            }
        }
        return $response;
    }

    public static function RemoveService($service){
        
    }

    public static function RemoveServices($starts){
        global $SYSTEMD;
        exec("systemctl list-units --type=service --all | grep $starts-", $services);
        foreach ($services as $svcLine) {
            $svcName = self::findServiceInList( explode(" ",$svcLine) );
            logc("[.] Stopping $svcName");
            exec("sudo systemctl stop $svcName > /dev/null 2>&1");
            //shell_exec("sudo systemctl stop $svcName");
            logc("[.] Disabling $svcName");
            exec("sudo systemctl disable $svcName > /dev/null 2>&1");
            //shell_exec("sudo systemctl disable $svcName");
        }

        $files = glob($SYSTEMD . "$starts*.service");
        if (!$files) {
            logc( "[x] Silinecek $starts* servis dosyası bulunamadı.","frontYellow");
        }
        foreach ($files as $file) {
            if (is_file($file)) {
                unlink($file); // dosyayı sil
                logc("[+] Silindi → " . basename($file), "frontGreen" );
            }
        }
        $services = [];

        exec("systemctl daemon-reload");
        logc("[+] systemctl daemon-reload çalıştırıldı","frontGreen");

    }

    public static function CreateRouters($configFile,$starts){
        global $SYSTEMD;
        if (!file_exists($configFile)) {
            return logc("[x] config.json bulunamadı [$configFile]!","backRed");
        }

        $config = json_decode(file_get_contents($configFile), true);
        if (!$config) {
            return logc("[x] config.json JSON formatına uygun değil [$configFile]!","backRed");
        }

        $user = "root";
        $file = basename($configFile);
        $serviceName = "$starts.service";
        $servicePath = $SYSTEMD . $serviceName;
        $workdir = "/web/server/router/";
            // ---------- SERVİS DOSYASI ÜRET ----------
$service = <<<SERVICE
[Unit]
Description=$file ROUTER SUNUCUSU
After=network.target

[Service]
ExecStart=/usr/local/go/bin/go run main.go config="$configFile"
Restart=always
User=$user
WorkingDirectory=$workdir

[Install]
WantedBy=multi-user.target
SERVICE;

        if (file_put_contents($servicePath, $service) === false) {
            logc("[x] SERVİS DOSYASI YAZILAMADI → $servicePath (root ile çalıştır!)", "backRed");
            return;
        }


        // ---------- SYSTEMCTL İŞLEMLERİ ----------
        exec("systemctl daemon-reload");
        exec("systemctl enable $serviceName > /dev/null 2>&1");
        exec("systemctl restart $serviceName > /dev/null 2>&1");
    
        logc("[+] $serviceName → Router", "frontGreen");

    }



    public static function CreateServices($configFile,$starts){
        global $SYSTEMD;
        $user = "root";
        if (!file_exists($configFile)) {
            return logc("[x] config.json bulunamadı [$configFile]!","backRed");
        }

        $config = json_decode(file_get_contents($configFile), true);
        if (!$config) {
            return logc("[x] config.json okunamadı veya hatalı JSON!","backRed");
        }


        // === ANA DÖNGÜ ===
        foreach ($config as $ports => $data) {
            foreach(ranges($ports) as $port){
                $domain = $data["domain"];
                logc("$domain:$port", "bold");
                $description = $data["description"];

                // go / php / httpd anahtarını bul
                $type = null;
                $path = null;
                foreach (["go", "php", "httpd","node", "sh", "bash"] as $t) {
                    if (isset($data[$t])) {
                        $type = $t;
                        $path = $data[$t];
                        break;
                    }
                }
                if (!$type) {
                    logc("[x] $domain için geçerli servis türü bulunamadı! (go/php/bash/sh/httpd yok)", "backRed");
                    continue;
                }

                $serviceName = "$starts-{$domain}-{$port}.service";
                $servicePath = "$SYSTEMD/$serviceName";

                // ---------- PORT TEMİZLE ----------
                exec("fuser -k {$port}/tcp 2>/dev/null");
                
                logc("  [+] Port temizlendi → $port");

                // ---------- EXECSTART OLUŞTUR ----------
                switch ($type) {

                    case "go":
                        $workdir = $path;
                        $exec = "/usr/local/go/bin/go run main.go port={$port}";
                        shell_exec("cd $path;/usr/bin/go mod init app > /dev/null 2>&1");
                        shell_exec("cd $path;/usr/bin/go mod tidy > /dev/null 2>&1");
                        break;

                    case "node":
                        $workdir = $path;
                        $exec = "/usr/local/bin/node main.js port={$port}";
                        shell_exec("cd $path;npm install;");
                        break;

                    case "sh":
                        $workdir = $path;
                        $exec = "/bin/bash main.sh --port {$port}";
                        break;

                    case "bash":
                        $workdir = $path;
                        $exec = "/bin/bash main.sh --port {$port}";
                        break;


                    case "php":
                        $workdir = $path;
                        $exec = "/usr/bin/php -d upload_max_filesize=500M -d post_max_size=500M -S 0.0.0.0:{$port} -t {$path} {$path}/router.php";
                        break;

                    case "httpd":
                        $workdir = $path;
                        $httpdConf = "/tmp/httpd-{$port}.conf";
                        file_put_contents($httpdConf, "
            ServerRoot \"/usr\"
            Listen {$port}
            DocumentRoot \"$path\"
            <Directory \"$path\">
                AllowOverride All
                Require all granted
            </Directory>
            ");

                        $exec = "/usr/sbin/httpd -DFOREGROUND -f $httpdConf";
                        break;
                }

// ---------- SERVİS DOSYASI ÜRET ----------
$service = <<<SERVICE
[Unit]
Description=$description
After=network.target

[Service]
ExecStart=$exec
Restart=always
User=$user
WorkingDirectory=$workdir

[Install]
WantedBy=multi-user.target
SERVICE;

                if (file_put_contents($servicePath, $service) === false) {
                    logc("  [x] SERVİS DOSYASI YAZILAMADI → $servicePath (root ile çalıştır!)", "backRed");
                    continue;
                }

                

                // ---------- SYSTEMCTL İŞLEMLERİ ----------
                exec("systemctl daemon-reload");
                exec("systemctl enable $serviceName > /dev/null 2>&1");
                exec("systemctl restart $serviceName > /dev/null 2>&1");
                logc("  [+] Servis oluşturuldu: ");
                logc("      $serviceName");
            }
        }

        logc("[+] Tüm servisler başarıyla oluşturuldu!", "frontGreen");

    }


    public static function TestServices($configFile,$starts){
        global $SYSTEMD;
        if (!file_exists($configFile)) {
            return logc("[x] config.json bulunamadı!","backRed");
        }

        $config = json_decode(file_get_contents($configFile), true);
        if (!$config) {
            return logc("[x] config.json okunamadı veya hatalı JSON!","backRed");
        }

        foreach ($config as $ports => $data) {
            foreach(ranges($ports) as $port){
                $domain = $data["domain"];
                
                $testUrl = "http://localhost:$port";
                $cmd = sprintf(
                    'curl -s -o /dev/null -w "%%{http_code}" "%s"',
                    $testUrl
                );
                $httpCode = trim(shell_exec($cmd));
                $httpCode = intval($httpCode);

                $serviceName = "$starts-{$domain}-{$port}.service";
                $servicePath = "$SYSTEMD/$serviceName";

                $info = "$domain:$port";

                if ($httpCode === "000") {
                    $command = "journalctl -u $serviceName -n 4";
                    logc($command,"33");
                    echo shell_exec($command);
                    logc("[x] Test başarısız → $info (Sunucu cevap vermedi)", "31");
                } elseif ((int)$httpCode >= 200 && (int)$httpCode < 400) {
                    logc("[+] Test başarılı → $info → HTTP $httpCode", "frontGreen");
                } else {
                    $command = "journalctl -u $serviceName -n 4";
                    logc($command,"33");
                    echo shell_exec($command);
                    logc("[x] Test hatalı yanıt → $info → HTTP $httpCode", "31");
                }
            }
        }
    }


    public static function Install($folder){
        self::CreateServices("/web/config/$folder/services.json",name($folder,"service"));
        self::CreateRouters("/web/config/$folder/routers.json",name($folder,"router"));
    }
    public static function Remove($folder){
        self::RemoveServices(name($folder,"service"));
        self::RemoveServices(name($folder,"router"));
    }
    public static function Reset($folder){
        self::Remove($folder);
        self::Install($folder);
    }
    public static function Test($folder){
        self::TestServices("/web/config/$folder/services.json",preg_replace('/[^a-zA-Z0-9_-]/', '', $folder)."-"); 
    }

    public static function Restart($folder){
        self::RemoveServices(name($folder,"router"));
        self::CreateRouters("/web/config/$folder/routers.json",name($folder,"router"));
    }

    public static function Stop($folder){
        self::RemoveServices(name($folder,"router"));
    }

    public static function Start($folder){
        self::CreateRouters("/web/config/$folder/routers.json",name($folder,"router"));
    }
    

    















    /*
        $domain : example.com
        $port   : 20900
        $serve  : go:/web/server/terminal
        $description : This is a terminal app
    */
    //+++++++++++++++++++++++++++
    public static function CreatePort($starts,$domain,$port,$serve,$description="",$user="root"){
        global $SYSTEMD;

        //$serviceName = "$starts-".slugify($domain)."-{$port}.service";
        //$serviceName = "$starts-{$port}.service";
        $serviceName = slugify("$starts")."-{$port}.service";
        $servicePath = "$SYSTEMD/$serviceName";
        logc(">>> {$domain}:{$port}");

        if(count(explode(":",$serve))<2) return logc("[x] $domain için serve($serve) doğru bir yapılandırma değil");


        $lang = explode(":",$serve)[0];
        $path = explode(":",$serve)[1];
        $workdir = $path;

        $exec = "echo 'ERROR'";
        if($lang=="go"){
            $exec = "/usr/local/go/bin/go run main.go port={$port}";
            shell_exec("cd $path;/usr/bin/go mod init app > /dev/null 2>&1");
            shell_exec("cd $path;/usr/bin/go mod tidy > /dev/null 2>&1");
        }else if($lang=="node"){
            $exec = "/usr/local/bin/node main.js port={$port}";
            shell_exec("cd $path;npm install;");
        }else if($lang=="sh" || $lang=="bash"){
            $exec = "/bin/bash main.sh --port {$port}";
        }else if($lang=="php"){
            $exec = "/usr/bin/php -d upload_max_filesize=500M -d post_max_size=500M -S 0.0.0.0:{$port} -t {$path} {$path}/router.php";
        }


// ---------- SERVİS DOSYASI ÜRET ----------
$service = <<<SERVICE
[Unit]
Description=$description
After=network.target
[Service]
ExecStart=$exec
Restart=always
User=$user
WorkingDirectory=$workdir
[Install]
WantedBy=multi-user.target
SERVICE;
// ---------- SERVİS DOSYASI ÜRET ----------

        if (file_put_contents($servicePath, $service) === false) {
            logc("  [x] SERVİS DOSYASI YAZILAMADI → $servicePath (root ile çalıştır!)", "backRed");
            return;
        }
        exec("systemctl daemon-reload");
        exec("systemctl enable $serviceName > /dev/null 2>&1");
        //@ PORT TEMİZLE
        exec("fuser -k {$port}/tcp 2>/dev/null");
        logc("  [+] Port temizlendi → $port");
        exec("systemctl restart $serviceName > /dev/null 2>&1");
        logc("  [+] Servis oluşturuldu: ");
        logc("      $serviceName");
    
    }


    


    public static function News($folder){
        
    }

    public static function Olds($folder){

    }

    public static function Update($folder){
        //self::RemoveServices(name($folder,"router"));
        //self::CreateRouters("/web/config/$folder/routers.json",name($folder,"router"));
    }

    //!!! HATALI
    public static function State($folder){
        foreach(DATA["routes"] as $domain => $route){
            $ports = ranges($route["ports"]);
            foreach($ports as $port){
                $service = service($folder,$domain,$port);
                echo shell_exec("systemctl status $service");
            }
        }
    }

    
    public static function ServiceInfo($pid){
        $o=shell_exec("systemctl status $pid 2>/dev/null");
        $name=$active=$mem=$cpu=$since=$min=null;
        foreach(explode("\n",$o)as$l){
            if(!$name&&preg_match('/● ([\w\-.@]+\.service)/',$l,$m))$name=$m[1];
            if(preg_match('/Active:\s+(\w+).*since\s+(.+?);/',$l,$m)){$active=$m[1];$since=$m[2];}
            if(!$mem&&preg_match('/Memory:\s+([\w\.,]+)/',$l,$m))$mem=$m[1];
            if(!$cpu&&preg_match('/CPU:\s+([\w\.,]+)/',$l,$m))$cpu=$m[1];
        }
        if($since){$t=strtotime($since);if($t)$min=floor((time()-$t)/60);}
        return ["pid"=>$pid,"service"=>$name,"active"=>$active,"memory"=>$mem,"cpu"=>$cpu,"since"=>$since,"uptime"=>$min];
    }



    public static function PID($port){
        $pid = @trim(shell_exec("ss -tulnp|awk '/:$port /&&/pid=/{match(\$0,/pid=([0-9]+)/,m);print m[1]}'"));
        if(!$pid) return 0;
        return intval($pid);
    }


    public static function Using($port){
        $pid = self::PID($port);
        if($pid){
            $info = self::ServiceInfo($pid);
            $info["port"] = $port;
            return $info;
        }else{
            logc("PORT($port) not active!","frontRed");
        }
        return false;
    }

    public static function Kill($port){
        $pid = self::PID($port);
        if($pid){
            $info = self::ServiceInfo($pid);
            table($info);
        }else{
            logc("PORT($port) not active!","frontRed");
            return;
        }
        $name = $info["service"];
        logc("[.] Stopping $name");
        exec("sudo systemctl stop $name > /dev/null 2>&1");
        logc("[.] Disabling $name");
        exec("sudo systemctl disable $name > /dev/null 2>&1");
        $file = SYSTEMD."$name";
        if (is_file($file)) {
            unlink($file); // dosyayı sil
            logc("[+] Removed $name", "frontGreen" );
        }
    }

    //public static function RouteInfo()

    public static function Info(){
        $PORTS = [];
        foreach(DATA["routes"] as $domain => $route){
            if(!array_key_exists("serve",$route)){
                //continue;
            }
            $ports = [];
            if(isset($route['ports'])){
                $ports = ranges($route["ports"]);
            }
            $route["domain"] = $domain;
            foreach($ports as $port){
                $route["port"] = $port;
                $PORTS[$port] = $route;
            }
        }

        foreach($PORTS as $i => $route){
            $pid = self::PID($route['port']);
            $PORTS[$i]["pid"] = $pid;
            $PORTS[$i]["active"] = $pid > 0 ? "active" : "";
            $PORTS[$i]["service"] = "";
            $PORTS[$i]["description"] = @$PORTS[$i]["description"] || "";
            
            if($pid){
                $service_info = self::ServiceInfo($pid);
                $PORTS[$i]["service"] = $service_info['service'];
                $PORTS[$i]["memory"] = $service_info['memory'];
                $PORTS[$i]["cpu"] = $service_info['cpu'];
                $PORTS[$i]["uptime"] = $service_info['uptime'];
            }
        }

        return $PORTS;
    }

    public static function GetInfo(){
        $routers = [Service::Using(DATA['http']),Service::Using(DATA['https'])];
        $routers[0]["domain"]="[http]";
        $routers[1]["domain"]="[https]";
        $info = [...array_values(Service::Info()),...$routers];
        return $info;
    }

    public static function WriteInfo(){
        $info = self::GetInfo();
        table(array_values($info),"pid,port,domain,active,service,memory,cpu,uptime");
    }

    public static function Up($port){
        $PORTS = self::Info();
        foreach($PORTS as $i => $route){
            if($route['port']==$port){
                if($route["pid"]==0 ){
                    if(array_key_exists("serve",$route)){
                        logc("[.] ".$route["port"] ." up starting");
                        self::CreatePort("kuruyo-".NAME."",$route['domain'],$route['port'],$route['serve'],$route['description'],$user="root");
                    }else{
                        logc("[x] ".$route['domain'].":".$route['port']." has not serve","frontRed");
                    }
                }else{
                    logc("[+] ".$route["port"] ." is already active" );
                }
            }
        }
    }
    

    public static function Ups(){
        $PORTS = self::Info();
        foreach($PORTS as $i => $route){
            if($route["pid"]==0 ){
                if(array_key_exists("serve",$route)){
                    logc("[.] ".$route["port"] ." upping");
                    self::CreatePort("kuruyo-".NAME."",$route['domain'],$route['port'],$route['serve'],$route['description'],$user="root");
                }else{
                    logc("[x] ".$route['domain'].":".$route['port']." has not serve","frontRed");
                }
            }else{
                logc("[+] ".$route["port"] ." is already up" );
            }
        }
    }
    



    public static function Work(){
        global $SYSTEMD;
        
        $file   = NAME;
        $config = FILE;
        
        $user = "root";
        $serviceName = "kuruyo-{$file}.service";
        $servicePath = $SYSTEMD . $serviceName;
        $workdir = "/web/server/engine/";
            // ---------- SERVİS DOSYASI ÜRET ----------
$service = <<<SERVICE
[Unit]
Description=$file ROUTER SUNUCUSU
After=network.target

[Service]
ExecReload=/bin/kill -10 \$MAINPID
ExecStart=/usr/local/go/bin/go run main.go config="$config"
Restart=always
User=$user
WorkingDirectory=$workdir

[Install]
WantedBy=multi-user.target
SERVICE;

        if (file_put_contents($servicePath, $service) === false) {
            logc("[x] Service file doesnt writed ($servicePath) (run as root!)", "backRed");
            return;
        }
        // ---------- SYSTEMCTL İŞLEMLERİ ----------
        exec("systemctl daemon-reload");
        exec("systemctl enable $serviceName > /dev/null 2>&1");
        exec("systemctl restart $serviceName > /dev/null 2>&1");
        logc("[+] Activated $serviceName", "frontGreen");
    }



    public static function Status($folder){
        //sudo systemctl status mysql
        global $SYSTEMD;
        $starts = name($folder,"service");
        $configFile = "/web/config/$folder/services.json";
        if (!file_exists($configFile)) {
            return logc("[x] config.json bulunamadı!","backRed");
        }

        $config = json_decode(file_get_contents($configFile), true);
        if (!$config) {
            return logc("[x] config.json okunamadı veya hatalı JSON!","backRed");
        }

        foreach ($config as $ports => $data) {
            foreach(ranges($ports) as $port){
                $domain = $data['domain'];
                echo "\n===============================\n";
                logc( "|  ". $domain . " : " . $port ,"frontYellow");
                echo "===============================\n";

                $serviceName = "$starts-{$domain}-{$port}";
                echo join("\n  ",explode("\n",shell_exec("systemctl status ".$serviceName )));
            }
        }

        $starts = name($folder,"router");
        echo "\n===============================\n";
        echo "|         ROUTER              |\n";
        echo "===============================\n";
        $serviceName = "$starts";
        echo join("\n  ",explode("\n",shell_exec("systemctl status ".$serviceName )));

    }

    public static function Log($folder){
        //self::bash("whoami");
        self::bash("journalctl -u ".name($folder)." -f");
    }

}





//Service::Remove("router-kuruyo");
//Service::CreateRouters("router-kuruyo");
//Service::Remove("kuruyo");
//Service::CreateServicess("./config.json","kuruyo");
//Service::Test("./config.json","kuruyo");

$args = $_SERVER['argv'];
array_shift($args);


if(count($args)<2){
    echo "\n";
    echo "  Usage: \n";
    //echo "    install [FOLDER]  - install services and routers\n";
    //echo "    remove  [FOLDER]  - remove service and routers \n";
    //echo "    reset   [FOLDER]  - reset service and routers \n";
    //echo "    update  [FOLDER]  - update services after router \n";
    //echo "    news    [FOLDER]  - install news services \n";
    //echo "    olds    [FOLDER]  - remove olds services \n";

    
    //echo "-----------------------------------------\n";
    //echo "    start   [FOLDER]  - start routers \n";
    //echo "    stop    [FOLDER]  - stop routers \n";
    //echo "    restart [FOLDER]  - restart routers \n";
    //echo "-----------------------------------------\n";
    //echo "    actives [FOLDER]  - active servers and routers\n";
    //echo "    state   [FOLDER]  - current status of folder \n";
    //echo "    status  [FOLDER]  - current status of folder \n";
    //echo "    test    [FOLDER]  - current status of folder \n";
    //echo "    log     [FOLDER]  - journalctl of router\n";
    //echo "-----------------------------------------\n";
    echo "    kill      [PORT]               - remove this port using service\n";
    echo "    up        [FOLDER] [ PORT ]    - up services on this file with port \n";
    echo "    reup      [FOLDER] [ PORT ]    - re-up port on this file\n";
    echo "    ups       [FOLDER]             - up services on this file \n";
    echo "    work      [FOLDER]             - work router\n";
    echo "    info      [FOLDER] (json)      - list services info\n";
    echo "    info      [ PORT ] (json)      - get port info\n";
    //echo "+   reload  [FOLDER]  - router reloading\n";
    


    echo "\n";
    die();
}



$action = $args[0];
$param  = $args[1];

if($action=="using" || $action=="pid"  || $action=="kill" || ($action=="info" && is_numeric($param))){

}else{
    //+ FILE AREA
    define("NAME", $param);
    define("FILE","/web/config/".$param."/routers.json");
    define("JSON", @file_get_contents(FILE));
    if(!JSON){
        logc("[x] routers.json dosyası bulunamadı! [$FILE]!","backRed");
        die();
    }
    define("DATA", json_decode(JSON, true));
    if (!DATA) {
        logc("[x] config.json JSON formatına uygun değil [$FILE]!","backRed");
        die();
    }
}


/*

if($action=="install"){
    Service::Install($param);
}

if($action=="remove"){
    Service::Remove($param);
}

if($action=="reset"){
    Service::Reset($param);
}

if($action=="update"){
    Service::Update($param);
}
    
if($action=="start"){
    Service::Start($param);
}
    
if($action=="stop"){
    Service::Stop($param);
}
if($action=="restart"){
    Service::Restart($param);
}

if($action=="test"){
    Service::Test($param);
}

if($action=="status"){
    Service::Status($param);
}

if($action=="state"){
    Service::State($param);
}

if($action=="log"){
    Service::Log($param);
}

if($action=="actives"){
    print_r(Service::ActivePorts($param));
}

if($action=="news"){
    Service::News($param);
}

if($action=="olds"){
    Service::Olds($param);
}

*/




/*


if($action=="services"){
    if(@$args[2]!="json"){
        table(array_values(Service::Info()),"pid,port,domain,active,service,memory,cpu,uptime");
    }else{
        json(Service::Info());
    }
}

if($action=="routers"){
    $info = [Service::Using(DATA['http']),Service::Using(DATA['https'])];
    if(@$args[2]!="json"){
        table(array_values($info));
    }else{
        json($info);
    }
}
*/

if($action=="info"){
    if(is_numeric($param)){
        $info = Service::Using($param);
        if(!$info) die();
        if(@$args[2]=="json"){
            json($info);
        }else{
            table($info,"pid,port,domain,active,service,memory,cpu,uptime");
        }
    }else{
        $info = Service::GetInfo();
        if(@$args[2]!="json"){
            table(array_values($info),"pid,port,domain,active,service,memory,cpu,uptime");
        }else{
            json($info);
        }
    }
}


if($action=="kill"){
    Service::Kill($param);
}


if($action=="up"){
    $port = @$args[2];
    if(!$port) die(logc("[X] kuruyo up [FOLDER] [PORT] port is empty","frontRed"));
    Service::Up($port);
    Service::WriteInfo();
}

if($action=="reup"){
    $port = @$args[2];
    if(!$port) die(logc("[X] kuruyo up [FOLDER] [PORT] port is empty","frontRed"));
    Service::Kill($port);
    Service::Up($port);
    Service::WriteInfo();
}


if($action=="ups"){
    Service::Ups();
}




if($action=="work"){
    Service::Work();
}


if($action=="reload"){
    $info = Service::Using(DATA['http']);
    if(!$info) die();
    shell_exec("systemctl restart ".$info['service']);
    logc("[+] Restarted ".$info['service'],"frontGreen");
    Service::WriteInfo();
}