<?php
error_reporting(E_ALL);
ini_set('display_errors', 1);


$backUPConfig = "/web/config/backup.config.json";
$filesBase = "/web";

// --- CONFIG LOAD ---
// --- CONFIG LOAD ---
if (!file_exists($backUPConfig)) {
  $config = [];
}else{
  $config = json_decode(file_get_contents($backUPConfig), true);
  if(!$config) $config=[];
}

// --- TOKEN AUTO ROTATION (daily) ---
$today = date("Y-m-d");
if (!isset($config['token']) || !isset($config['updated']) || $config['updated'] !== $today) {
    $config['token'] = bin2hex(random_bytes(32));
    $config['updated'] = $today;
    file_put_contents($backUPConfig, json_encode($config, JSON_PRETTY_PRINT));
}

$TOKEN = $config['token'];


// --- DETECT API CALLS ---
$action = $_GET['action'] ?? null;

// =====================================================
// === ACTION: LIST DIRECTORY ==========================
// =====================================================

if ($action === "list") {
    $auth = getBearerToken();
    if ($auth !== $TOKEN) {
        http_response_code(403);
        die("Token hatalı");
    }

    $path = $_GET['path'] ?? "";
    $full = realpath($filesBase . "/" . $path);
        

    if (!$full || strpos($full, $filesBase) !== 0) {
        http_response_code(403);
        die("\nGeçersiz path");
    }

    $list = scandir($full);
    $output = "";

    foreach ($list as $file) {
        if ($file === "." || $file === "..") continue;

        $p = $full . "/" . $file;


        if (is_dir($p)) {
            $output .= "0 " . $file . "/\n";
        } else {
            $output .= md5_file($full."/".$file) . " " . $file . "\n";
        }
    }

    header("Content-Type: text/plain");
    echo $output;
    exit;
}


// =====================================================
// === ACTION: DOWNLOAD FILE ===========================
// =====================================================
if ($action === "download") {
    $auth = getBearerToken();
    if ($auth !== $TOKEN) {
        http_response_code(403);
        die("Token hatalı");
    }

    $path = $_GET['path'] ?? "";
    $full = realpath($filesBase . "/" . $path);

    if (!$full || strpos($full, $filesBase) !== 0) {
        http_response_code(403);
        die("Geçersiz path");
    }

    header("Content-Type: application/octet-stream");
    header("Content-Disposition: attachment; filename=\"" . basename($full) . "\"");
    readfile($full);
    exit;
}



$IP = exec("curl -s https://ifconfig.me");
$PORT = count(explode(":", $_SERVER['HTTP_HOST'])) > 1 
    ? explode(":", $_SERVER['HTTP_HOST'])[1] 
    : "";
if($PORT=="") $PORT = $_SERVER["REQUEST_SCHEME"]=="https" ? 443 : 80;

// =====================================================
// === NO ACTION → GENERATE curl.sh =====================
// =====================================================



$HOST_CMD = "-H \"Host: ". explode(":",$_SERVER['HTTP_HOST'])[0] ."\"";

$curlScript = <<<BASH
#!/bin/bash
SERVER="https://${IP}:${PORT}"
TOKEN="$TOKEN"
OUTDIR="./"

mkdir -p "\$OUTDIR"

download_dir() {
    local path="$1"

    #echo "🔎︎ \$path"
    echo "OO \$path"

    # LIST isteği
    list=$(curl -k -s -X POST -H "Authorization: Bearer \$TOKEN" "\$SERVER/?action=list&path=\$path" $HOST_CMD )



    # Satır satır işlem
    while IFS= read -r line; do
        [ -z "\$line" ] && continue

        # MD5 ve dosya adını ayır
        local md5=\${line%% *}
        local name=\${line#* }
        full="\$path/\$name"
        local local_file="\$OUTDIR/\$full"
        local local_md5="0"
        if [[ -f \$local_file ]]; then
            # md5sum –b returns “<md5> <filename>”; cut out the first field.
            local_md5=\$(md5sum -b "\$local_file" | awk '{print \$1}')
        fi
        
        
        if [[ -e "\$full" && "\$name" == ~* ]]; then
            echo "~~  \$full"
            continue
        fi

        # klasör mü? slash ile bitiyor
        if [[ "\$name" == */ ]]; then
            clean="\${name%\/}"     # sondaki slash'i kaldır
            #echo "🗀  \$path/\$clean"
            echo "|-  \$path/\$clean"
            mkdir -p "\$OUTDIR/\$path/\$clean"
            download_dir "\$path/\$clean"
        else
            if [[ "\$md5" == "\$local_md5" ]]; then
                #echo "⇋  \$full"
                echo "==  \$full"
            else
                #echo "🗎  \$full"
                echo ">>  \$full"
                curl -k -s -H "Authorization: Bearer \$TOKEN" \
                    "\$SERVER/?action=download&path=\$full" \
                    $HOST_CMD \
                    -o "\$OUTDIR/\$full"
            fi
        fi
    done <<< "\$list"
}

download_dir "$1"
BASH;

//file_put_contents(".curl.sh", $curlScript);

if($action==="start"){
    $auth = trim( getBearerToken() );
    
    if ($auth !== $TOKEN) {
        http_response_code(403);
        die('echo "PERMISSION DENIED"');
    }
    echo $curlScript;
    die();
}


// =====================================================
// === HELPER FUNCTION ================================
// =====================================================
function getBearerToken() {
    $hdr = getallheaders();
    if (!isset($hdr['Authorization'])) return null;
    if (preg_match('/Bearer (.+)/', $hdr['Authorization'], $m)) {
        return $m[1];
    }
    return null;
}


?>NO REQUEST
