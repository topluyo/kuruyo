<?php
error_reporting(E_ALL);
ini_set('display_errors', 1);


$backUPConfig = "/web/config/backup.config.json";
$filesBase = "/web";

// --- CONFIG LOAD ---
if (!file_exists($backUPConfig)) {
  $config = [];
}else{
  $config = json_decode(file_get_contents($backUPConfig), true);
  if(!$config) $config=[];
}
$today = date("Y-m-d");
// --- TOKEN AUTO ROTATION (daily) ---
if (!isset($config['token']) || !isset($config['updated']) || $config['updated'] !== $today) {
  $config['token'] = bin2hex(random_bytes(32));
  $config['updated'] = $today;
  file_put_contents($backUPConfig, json_encode($config, JSON_PRETTY_PRINT));
}
$TOKEN = $config['token'];


function bash($command) {
  $output = [];
  $return_var = 0;
  exec($command . " 2>&1", $output, $return_var);
  return implode("\n", $output);
}

?>
<script src="//hasandelibas.github.io/documenter/documenter.js"></script>
<meta charset="utf8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<header body-class="show-menu theme-dark tab-system">
  <div title="">🐺 <?= $_SERVER['HTTP_HOST'] ?></div>
  <div class="space"></div>
</header>


<style>
[safe]{
  padding: .6em 1.2em;
  text-decoration: none;
  color: var(--front);
  border-radius: .25em;
  background: #8882;
  overflow:hidden;
  display:inline-block;
}
[fix]{
  display: grid;
  gap:2em;
  grid-template-columns: repeat(auto-fit, minmax(var(--fix-width), 1fr));
}
</style>

# Main
<div fix style="--fix-width:100px;">
  <a safe hover target="_blank" href="/master/root">📁 Master</a>
  <a safe hover target="_blank" href="/adminer">🗄️ Database</a>
  <a safe hover target="_blank" href="/goeditor">💻 GoEditor</a>  
</div>


# Database

<?php 



if(@$_GET["action"]=="database-remove"){
  echo bash('mysql -u root -e "DROP DATABASE db"');
  echo bash('mysql -u root -e "DROP DATABASE log"');
}

if(@$_GET["action"]=="database-store"){
  @unlink("/web/database/db.sql");
  @unlink("/web/database/log.sql");
  echo "```";
  bash("cd /web/database; mysqldump -u root db > db.sql;");
  bash("cd /web/database; mysqldump -u root log > log.sql;");
  echo "✅ Yedek Alındı";
  echo "```";
  if( is_file("/web/database/db.sql") ){
    echo "<script>setTimeout(e=>documenter.message('✅ db Yedek Alındı'),1000)</script>\n";
  }
  if( is_file("/web/database/log.sql") ){
    echo "<script>setTimeout(e=>documenter.message('✅ log Yedek Alındı'),1000)</script>\n";
  }
}

if(@$_GET["action"]=="database-install"){
  echo "<textarea readonly spellcheck=off style='font-family:monospace;width:100%;margin-top:1em;height:20em;'>";

$cmd = <<<SQL
mysql -u root -e "
  CREATE DATABASE IF NOT EXISTS db
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;
  CREATE DATABASE IF NOT EXISTS log
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;
  CREATE USER IF NOT EXISTS 'master'@'localhost' IDENTIFIED BY 'master';
  GRANT ALL PRIVILEGES ON db.* TO 'master'@'localhost';
  GRANT ALL PRIVILEGES ON log.* TO 'master'@'localhost';
  FLUSH PRIVILEGES;
"
SQL;
  echo bash($cmd);
  echo bash("sed -i.bak 's/utf8mb4_0900_ai_ci/utf8mb4_unicode_ci/g' /web/database/db.sql");
  echo bash("mysql -u root db < /web/database/db.sql");
  echo bash("mysql -u root log < /web/database/log.sql");
  echo "</textarea>";
  echo "<script>setTimeout(e=>documenter.message('✅ Yedek Kuruldu'),1000)</script>\n";
}


?>
<div fix style="--fix-width:100px;">
  <a safe hover target="_blank" href="?action=database-store#Database">🔄 BackUp</a>
  <a safe hover target="_blank" href="?action=database-install#Database">🔄 Update</a>
  <div safe hover>⛔ ?action=database-remove</div>
  
</div>





# 💾 BackUp

Get Token From ``<?= __DIR__ ?>/.config.json``

- ``~`` ile başlayan klasörler yada dosyalar sunucudan indirildikten sonra güncellenmez.
- ``.`` ile başlayan klasör yada dosyalar sunucudan indirilmez.


<div>
<br>
</div>
<?php
$IP = exec("curl -s https://ifconfig.me");
$PORT = count(explode(":", $_SERVER['HTTP_HOST'])) > 1 
    ? explode(":", $_SERVER['HTTP_HOST'])[1] 
    : "";
if($PORT=="") $PORT = $_SERVER["REQUEST_SCHEME"]=="https" ? 443 : 80;
?>
<textarea spellcheck="false" style="height: 9em; width: 100%; font-family: monospace; white-space: pre; box-shadow: none; font-size: 0.8em; border-radius: 0.5em;">
mkdir /web
apt install curl
curl -k -X POST "https://<?= $IP ?>:<?= $PORT ?>/?action=start" \
    -H "Host: <?= str_replace("main.","backup.", $_SERVER['HTTP_HOST']) ?>" \
    -H "Authorization: Bearer <?= $TOKEN ?>" > backup.bash
bash backup.bash
</textarea>



# 🔁 Router



Router sistemi için örnek json bloğu

``/web/config`` klasörünün altında olur. Klasör içinde ``~system`` ``.build`` ``.alfa`` gibi klasörler olabilir.

```
 - config/
 |--- ~system
    |--- router.json
    |--- services.json
 |--- .build
    |--- router.json
    |--- services.json
```

<div preview="Update Routers">
```bash
kuruyo install ~system
kuruyo update ~system
kuruyo remove ~system
kuruyo start ~system  
kuruyo stop ~system  
kuruyo restart ~system  

kuruyo status ~system
kuruyo test ~system
kuruyo log ~system
```
</div>

<div preview="routers.json">
```json
{
  "ip": "35.228.122.185",
  "http": 2086,
  "https": 2096,
  "routes": {
    "backup.google.bozkuruyo.com":{
      "description": "Bu servisin 'basic' levelinin içindeki rate limitlere göre çalışır.",
      "proxies": ["http://localhost:17000"],
      "levels": ["basic"]
    },
    "terminal.google.bozkuruyo.com":{
      "description": "Bu servise 'password' içerdeki belirlenen parola ile giriş yapılır. Pursaklar bölgesinde çalışır.",
      "proxies": ["http://localhost:18000"],
      "levels": ["pursaklar","password"]
    },
    "main.google.bozkuruyo.com":{
      "description": "Bu servise passwordda belirlenen veri ile giriş yapılabilir.",
      "ports": "19000+4",
      "levels": ["password"]
    }
  },
  "levels": {
    "password":{
      "token":"cf5425fe705f4a03a03f528c88d4f245cf5425fe705f4a03a03f528c88d4f245"
    },
    "ev":{
      "ips":[
        "88.235.215.99"
      ]
    },
    "pursaklar":{
      "ips":[
        "78.189.98.0/23",
        "78.168.152.0/21",
        "88.226.88.0/21",
        "88.240.88.0/21",
        "88.227.182.0/23",
        "88.247.80.0/22",
        "88.241.0.0/16"
      ]
    },
    "basic":{
      "rate":["10r 10s 20w","20r 10s 60w"]
    },
    "cloudflare":{
      "ips":[
        "173.245.48.0/20",
        "103.21.244.0/22",
        "103.22.200.0/22",
        "103.31.4.0/22",
        "141.101.64.0/18",
        "108.162.192.0/18",
        "190.93.240.0/20",
        "188.114.96.0/20",
        "197.234.240.0/22",
        "198.41.128.0/17",
        "162.158.0.0/15",
        "104.16.0.0/13",
        "104.24.0.0/14",
        "172.64.0.0/13",
        "131.0.72.0/22",
        "2400:cb00::/32",
        "2606:4700::/32",
        "2803:f800::/32",
        "2405:b500::/32",
        "2405:8100::/32",
        "2a06:98c0::/29",
        "2c0f:f248::/32"
      ]
    }
  }
}
```
</div>


# 🔁 Service

```json
{
  "19000-19004":{
    "domain" "name-of-service",
    "description": "Its running on 19000, 19001, 19002, 19003, 19004 ports. /path/to/folder/router.php needed"
    "php": "/path/to/folder"
  },
  "20000+5":{
    "domain":"name-of-service",
    "description": "Its running on 20000, 20001, 20002, 20003, 20004, 20005 ports. /path/to/folder/router.php needed",
    "php": "/path/to/folder"
  }
  "8080":{
    "domain": "name-of-service",
    "description": "Its running on 8080 port. GoLang arguments is port=8080. File name is main.go",
    "go": "/path/to/folder"
  }
}
```

# Task 
<textarea spellcheck="false" id="note" style="height: 270px;width: 100%;height: calc(100% - 6.5em);resize: none;overflow: auto;"></textarea>

<script>
// Run when the page loads
window.addEventListener("DOMContentLoaded", () => {
    const note = document.getElementById("note");

    // Load saved value (if any)
    const saved = localStorage.getItem("noteText");
    if (saved !== null) {
        note.value = saved;
    }

    // Save on change or typing
    note.addEventListener("input", () => {
        localStorage.setItem("noteText", note.value);
    });
});
</script>




# INFO
```
<?php print_r($_SERVER) ?>
```

<style>
[preview] pre{
  margin:0;
}
</style>