<?php 


$BASE = substr($_SERVER["REQUEST_URI"],strlen("/goeditor")) ."/";
$BASE = explode("?",$BASE)[0];
$BASE = str_replace("//","/",$BASE);
$BASE = $BASE . "/";
$BASE = str_replace("//","/",$BASE);



if($BASE=="" || $BASE=="/"){
  echo "<div padding>/goeditor/web/path/folder</div>";
}


/*
function slugify(string $text): string{
  $slug = preg_replace('/[^A-Za-z0-9]+/', '-', $text);
  $slug = preg_replace('/-+/', '-', $slug);
  $slug = trim($slug, '-');
  return $slug;
}
*/



function first($sql){
  $pdo = new PDO(
      "mysql:host=localhost;dbname=db;charset=utf8mb4",
      "master",
      "master",
      [
          PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
          PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC
      ]
  );
  $stmt = $pdo->prepare($sql);
  $stmt->execute();
  // Return result or null
  return $stmt->fetch(PDO::FETCH_ASSOC);
}



if(isset($_GET['new-folder'])){
  mkdir($BASE."/Source/".$_GET['new-folder']);
}

if(isset($_GET['new-file'])){
  echo $_GET["new-file"];
  $name = explode("/",$_GET['new-file'])[1];
  $file  = "func ".$name."(r *http.Request, param map[string]interface{}) Response {\n";
  $file .= "  return Success( 1 )\n";
  $file .= "}";
  file_put_contents($BASE."/Source/".$_GET['new-file'].".go",$file);
  die();
}

if(isset($_GET['save'])){
  file_put_contents($BASE."/Source/".$_POST['file'].".go","/*\n".$_POST["comment"]."*/\n".$_POST['code']);
  exit(1);
}

function isfloat($num) {
  if (is_float($num)) {
    return true;
  }
  if (is_numeric($num)) {
    return strpos((string)$num, '.') !== false;
  }
  return false;
}

function UpperCase($d) {
  return preg_replace_callback('/(^|_)([a-z0-9]+)/', function ($matches) {
    $word = ucfirst($matches[2]);
    if (strtolower($matches[2]) === 'id') {
      $word = 'ID';
    }
    if (strtolower($matches[2]) === 'ids') {
      $word = 'IDS';
    }
    return $word;
  }, $d);
}
if(isset($_GET['model'])){
  require_once "/web/sites/system/system.php";

  $_POST['data'] = str_replace("\r","",$_POST['data']);
  

  $name = $_POST['name'];   
  $sql  = $_POST['data'];

  preg_match_all("/([$][^\(]+)\(([^\)]+)\)/m",$sql,$sql_parameters_data);

  $sql_for_db = $sql;
  $sql_for_go = $sql;
  $sql_parameters = [];
  if($sql_parameters_data){
    foreach($sql_parameters_data[0] as $i=>$p){
      $sql_for_db = str_replace($sql_parameters_data[0][$i],$sql_parameters_data[2][$i],$sql_for_db);
      $parameter = substr($sql_parameters_data[1][$i],1);
      $sql_parameters[] = $parameter;
      $type = is_numeric($sql_parameters_data[2][$i]) ?  "int" : "string";
      if(is_float($sql_parameters_data[2][$i]) ) $type="float32";
      $sql_parameters_types[] = $type;
      $sql_for_go = str_replace($sql_parameters_data[0][$i],"?",$sql_for_go);
    }
  }

  $sql_for_go = str_replace("\"","\\\"",$sql_for_go);
  
  if(!preg_match("/LIMIT\s+\d+\s*$/",$sql_for_db)){
    $sql_for_db .= " LIMIT 1";
  }

  $model_type = explode("_",$name)[0];

  preg_match('/FROM\s+`?(\w+)`?/i', $_POST['data'], $matches);
  $tableName = $matches[1];
  

  $response = "";

  if($model_type=="SQL"){
    $sql_for_go = addslashes($sql_for_go);
    $response .= "func ".$name."(";
    foreach(array_unique($sql_parameters) as $i => $p){
      $response .= $p . " " . $sql_parameters_types[$i].", ";
    }
    $response .= ") int {\n";
    $response .= "  query := \"". join(" \" +\n    \"",explode("\n", $sql_for_go ))  ." \"\n";
    $response .= "  res,err := db.Exec(query,";

    foreach($sql_parameters as $i => $p){
      $response .= $p . ", ";
    }    

    $response .= ")\n";
    
    
    $response .="  if (err != nil) {\n";
    $response .="    log.Println(\"(".$name.") #SQL:  \", err)\n";
    $response .="    return 0\n";
    $response .="  }\n";
    
    $response .= "  _rows,err := res.RowsAffected()\n";
    $response .= "  if(err!=nil){ _rows=0 }\n";
    $response .= "  return int(_rows)\n";
    $response .="}";
    echo $response;
    exit();
	}


  $struct = [];

  $sql_for_db = preg_replace('/\r\n|\r|\n/', ' ', $sql_for_db);;
  //echo $sql_for_db;
  //write( $db->sql(str_replace('\n'," ",$sql_for_db)) );
  //exit();
  
  
  

  $data = first($sql_for_db);
  /*
  write($sql_for_db);
  if($data){ 
    $data = $data[0]; 
  }else{
    $sql_for_db = substr($sql_for_db,0,strlen($sql_for_db)-7);
    $data = $db->sql( $sql_for_db )[0]; 

  }
  */

  foreach($data as $k=>$d){
    $type = gettype($d);
    if($type=="integer") $type="int";
    if(isfloat($d)) $type="float32";
    
    $property = UpperCase($k);
    if(preg_match('/^[0-9]/', $property)) $property = "DATA_" . $property;
    $struct[$k] = [ $property , $type , "`db:\"".$k."\" json:\"".$k."\"`", $k];
  }

  
  if($model_type=="Model" || $model_type=="List"){
    $struct_name = "Struct_" . $name;
    $response = "type ".$struct_name." struct {\n";
    foreach($struct as $s){
      $response .= "  ".str_pad($s[0],24," ")."  ". str_pad($s[1],10," ")."  ". str_pad($s[2],24," ")."\n";
    }
    $response .= "}";
  }
  
  if($model_type=="Add"){

    $struct_name = "Struct_Get_" . UpperCase($tableName);
    $response = "type ".$struct_name." struct {\n";
    $response .= "  ".str_pad("ID",24," ")."  ". str_pad("int",10," ")."  ". str_pad("`db:\"id\" json:\"id\"`",24," ")."\n";
    foreach($struct as $s){
      if($s[0]=="ID") continue;
      $response .= "  ".str_pad($s[0],24," ")."  ". str_pad($s[1],10," ")."  ". str_pad($s[2],24," ")."\n";
    }
    $response .= "}\n\n";


    //@Add
    $response .= "func ".$name."(";
    foreach($struct as $s){
      if($s[0]=="ID") continue;
      $response .= $s[0]." ".$s[1].", ";
    }
    $response .= ") int {\n";


    $response .= "  res,err := db.Exec(\"INSERT INTO `".$tableName."` (";
    
    $i=0;
    foreach($struct as $s){
      if($s[0]=="ID") continue;
      $i++;
      $response .= "`".$s[3]."`,";
    }
    $response = substr($response,0,-1);
    
    $response .= ") VALUES (";
    $i=0;
    foreach($struct as $s){
      if($s[0]=="ID") continue;
      $i++;
      $response .= "?,";
    }
    $response = substr($response,0,-1);

    $response .= ")\",\n    ";

    $i=0;
    foreach($struct as $s){
      if($s[0]=="ID") continue;
      $i++;
      $response .= $s[0];
      $response .= ", ";
    }
    $response = substr($response,0,-1);

    $response .= ")\n\n";

    $response .= "  if(err!=nil){\n";
    $response .= "    log.Println(\"(".$name.") #SQL:  \", err)\n";
    $response .= "    return 0\n  }\n";
    $response .= "  _id,err := res.LastInsertId()\n";
    $response .= "  if(err!=nil){ _id=0 }\n";
    $response .= "  return int(_id)";
    $response .= "\n}\n\n";



    //@Set
    $response .= "func ". str_replace("Add_","Set_",$name) ."(id int,";
    foreach($struct as $s){
      if($s[0]=="ID") continue;
      $response .= $s[0]." ".$s[1].", ";
    }
    $response .= ") int {\n";

    $response .= "  res,err := db.Exec(\"UPDATE `".$tableName."` SET ";
    
    $i=0;
    foreach($struct as $s){
      if($s[0]=="ID") continue;
      $i++;
      $response .= "`".$s[3]."` = ?,";
    }
    $response = substr($response,0,-1);
    
    $response .= " WHERE `id`=?\",\n    ";
    
    $i=0;
    foreach($struct as $s){
      if($s[0]=="ID") continue;
      $i++;
      $response .= $s[0];
      $response .= ", ";
    }
    $response .= " id)\n\n";


    $response .= "  if(err!=nil){\n";
    $response .= "    log.Println(\"(".$name.") #SQL:  \", err)\n";
    $response .= "    return 0\n  }\n";
    $response .= "  _rows,err := res.RowsAffected()\n";
    $response .= "  if(err!=nil){ _rows=0 }\n";
    $response .= "  return int(_rows)";
    $response .= "\n}\n\n";


    //@Delete
    $response .= "func ".str_replace("Add_","Delete_",$name)."(id int) int {\n";
    $response .= "  res,err := db.Exec(\"DELETE FROM `".$tableName."` WHERE `id`=?\",id)\n";
    $response .= "  if(err!=nil){\n";
    $response .= "    log.Println(\"(".$name.") #SQL:  \", err)\n";
    $response .= "    return 0\n  }\n";
    $response .= "  _rows,err := res.RowsAffected()\n";
    $response .= "  if(err!=nil){ _rows=0 }\n";
    $response .= "  return int(_rows)\n";
    $response .= "}\n";


    //@Get
    $response .= "\nfunc ". str_replace("Add_","Get_",$name) ."(id int) *" . $struct_name ." {\n";
    
    $response .= "  var s ". $struct_name."\n";

    $__sql  = explode("WHERE",$_POST['data'])[0];
    $__sql .= " WHERE `id`=?";

    $response .= "  query := \"". join("\"+\n\"",explode("\n", $__sql ))  ." LIMIT 1\"\n";

    $response .= "  err := db.QueryRow(query, id).Scan(\n";
    //$response .= "    &s.ID,\n";
    foreach($struct as $s){
      $response .= "    &s.".$s[0].",\n";
    }
    $response .="  )\n";
    $response .= "  if(err!=nil){\n";
    $response .= "    log.Println(\"(".$name.") #SQL:  \", err)\n";
    $response .= "    return nil\n  }\n";
    $response .="  s.ID = id\n";
    $response .="  return &s\n";
    $response .="}";

  }

  if($model_type=="Model"){
    


    $response .= "\n\nfunc ".$name."(";
    foreach(array_unique($sql_parameters) as $i => $p){
      $response .= $p . " " . $sql_parameters_types[$i].", ";
    }
    $response .= ") (*".$struct_name.") {\n";
    $response .= "  var s ". $struct_name."\n";
    $response .= "  query := \"". join(" \"+\n\"",explode("\n", $sql_for_go ))  ." LIMIT 1\"\n";
    $response .= "  err := db.QueryRow(query,";
    

    foreach($sql_parameters as $i => $p){
      $response .= $p . ", ";
    }


    $response .= ").Scan(\n";
    foreach($struct as $s){
      $response .= "    &s.".$s[0].",\n";
    }
    $response .="  )\n";
    $response .="  if err != nil {\n";
    $response .="    if ( err != sql.ErrNoRows ) { log.Println(\"(".$name.") #SQL:  \", err) }\n";
    //$response .="    log.Println(\"(".$name.") #SQL:  \", err)\n";
    $response .="    return nil\n";
    $response .="  }\n";
    $response .="  return &s\n";
    $response .="}";
	}





  if($model_type=="List"){
    

    //$response .= "\n\nfunc ".$name."(id int) ([]".$struct_name.") {\n";
    $response .= "\n\nfunc ".$name."(";
    foreach(array_unique($sql_parameters) as $i => $p){
      $response .= $p . " " . $sql_parameters_types[$i].", ";
    }
    $response .= ") ([]".$struct_name.") {\n";

    $response .= "  var l []". $struct_name."\n";
    $response .= "  query := \"". join(" \" +\n    \"",preg_split("/\n/",$sql_for_go))  ."\"\n";
    $response .= "  rows, err := db.Query(query, ";
    foreach($sql_parameters as $i => $p){
      $response .= $p . ", ";
    }
    $response .= ")\n";
    $response .= "  defer rows.Close()\n";
    $response .= "  if err != nil {\n";
    $response .= "    log.Println(\"(".$name.") #SQL:  \", err)\n";
    $response .= "    return l\n  }\n";

    $response .= "  for rows.Next() {\n";
    $response .= "    var s " . $struct_name . "\n";
    $response .= "    err := rows.Scan(\n";
    foreach($struct as $s){
      $response .= "      &s.".$s[0].",\n";
    }
    $response .= "    )\n";
    $response .= "    if err == nil { l = append(l, s) }\n";
    $response .= "  }\n";
    $response .= "  return l\n";
    $response .= "}";
	}

  

  write($response);
  exit();
}


function comments($code) {
  preg_match_all('/\/\*([\s\S]*?)\*\//', $code, $matches);
  return $matches[1]; // array of comment sections
}

function apify($text){
  $text = substr($text,6);
  $text = str_replace("_","/",$text);
  $text = strtolower($text);
  return $text;
}


function parseDescriptionJson($text) {
    $lines = explode("\n", trim($text));
    $result = [];
    $i = 0;
    while ($i < count($lines)) {
        $description = trim($lines[$i]);
        $i++;

        // collect JSON block
        $jsonLines = [];
        $braceCount = 0;
        while ($i < count($lines)) {
            $line = $lines[$i];
            $braceCount += substr_count($line, "{");
            $braceCount -= substr_count($line, "}");
            $jsonLines[] = $line;
            $i++;
            if ($braceCount <= 0 && !empty($jsonLines)) {
                break;
            }
        }

        $json = trim(implode("\n", $jsonLines));
        if ($json === "") {
            $json = "{}";
        }

        $result[] = [
            "description" => $description,
            "json" => $json
        ];
    }
    return $result;
}


if(isset($_GET['build']) && $_GET['build']==1){

  $mid = "";
  $api = "";
  //$output = "";
  $output = file_get_contents($BASE."Main/main.go");
  //$output .= file_get_contents($BASE."_main.go");

  $routers = [];
  $gofiles = "";
  $title = "";
  $files = glob($BASE . "Source/*/*.go");
  sort($files);
  $init_codes = "";
  
  foreach ($files as $filename) {
    $name = str_replace($BASE."Source/","",$filename);
    $name = str_replace(".go","",$name);
    $_title = explode("/",$name)[0];
    $_name = explode("/",$name)[1];

    $source = file_get_contents($filename);
    $comments = comments($source);
    $description = @$comments[0];
    
    if(explode("_",$_name)[0]=="Init"){
      $init_codes  .= "  " . $_name . "()\n";
    }
    
    $code = $source;
    foreach($comments as $comment){
      $code = str_replace("/*".$comment."*/","",$code);
    }
    $code = "\n//# " . $name . $code."\n"; 
    $gofiles .= $code;
    if(strpos($_name, "Route_")===0){
      $routers[] = $_name;
      

$mid .= "  mux.HandleFunc(\"/!api/" . apify($_name) . "\", func(w http.ResponseWriter, r *http.Request) {\n";
$mid .= "    var request map[string]interface{}\n";
$mid .= "    err := json.NewDecoder(r.Body).Decode(&request)\n";
$mid .= "    if err != nil {\n";
$mid .= "      AnswerResponse(w, Error(0,\"Invalid JSON\"))\n";
$mid .= "      return\n";
$mid .= "    }\n";
$mid .= "    AnswerResponse(w, $_name (r, request) )\n";
$mid .= "  })\n";

      if($title!=$_title){
        $api .= "# ".$_title."\n";
        $title = $_title;
      }
      
      foreach(parseDescriptionJson($description) as $desc){
        $api .= "## ". apify($_name)."<space></space><div button='test'>Test</div>\n";
        $api .= $desc['description'] . "\n";
        $api .= "```\n/!api/".apify($_name)."\n";
        $api .= $desc['json']."\n```\n";
      }
    }
  }

  

  file_put_contents($BASE."Build/api.html", str_replace("//@CODE",$api,file_get_contents($BASE."Main/api.html")) );
  
  
  $router = "";
  $router .=  "\n\t\t\t" . "if 0==1 {  \n\t\t\t}";
  foreach($routers as $r){
    //$router .=  "\n\t\t\t" . 'if req.API=="/!api/'.apify($r).'" { response[index] = '.$r.'(r,req.Data) }';
    $router .=  'else if req.API=="/!api/'.apify($r).'"' . "{\n\t\t\t  response[index] = ".$r."(r,req.Data) \n\t\t\t}";
  }
  $router .=  'else{ response[index] = Error(0,"api not found") }';

  $output = str_replace("//@CODE",$router,$output);

  $output = str_replace("//@APIS",$mid,$output);

  $output = str_replace("//@INIT",$init_codes,$output);

  $output .= $gofiles;

  file_put_contents($BASE."Build/"."main.go",$output);

  
  //@ HATA TESPITI
  $lines = explode("\n",$output);
  chdir($BASE."Build/");
  $cmd = "/program/go build main.go 2>&1";
  exec($cmd, $error);

  $responseErrors = [];
  foreach($error as $k=>$o){
    if(substr($o,0,10)=="./main.go:"){
      $line = explode(":",substr($o,10))[0]-1;
      for($i=$line;$i>0;$i--){
        if(substr($lines[$i],0,3)=="//#"){
          $message = substr($lines[$i],4);
          $plusLine = $line - $i;
          $message = "<a href='#".explode("/",$message)[1]."'>".$message.":". $plusLine ."</a>";
          $responseErrors [] = $message . "\t" . explode(":",substr($o,3))[3] . "<small>".$o."</small>";
          //array_splice($error,$i,0,[$message]);
          //$error[$k] .= substr($lines[$i],3);
          break;
        }
      }
    }
  }
  
  //! BURAYA AUTO API GELECEK


  $response = implode("\n", $responseErrors);
  echo $response;
  
  exit();
}


if(isset($_GET['test'])){
  echo exec("sudo /usr/local/go/bin/go build main.go");
  exit();
}


function createService($name, $exec, $user = null, $workdir = null) {
  if ($user === null) {
      $user = get_current_user();
  }
  if ($workdir === null) {
      $workdir = getcwd();
  }

  $service = <<<SERVICE
[Unit]
Description=$name service
After=network.target

[Service]
ExecStart=$exec
Restart=always
User=$user
WorkingDirectory=$workdir

[Install]
WantedBy=multi-user.target
SERVICE;

  $file = "/etc/systemd/system/$name.service";

  // write service file (need sudo/root)
  file_put_contents($file, $service);

  // reload, enable and start
  shell_exec("systemctl stop $name.service");
  shell_exec("systemctl daemon-reload");
  shell_exec("systemctl enable $name.service");
  shell_exec("systemctl start $name.service");

  return "$name.service created and started";
}
if(isset($_GET["start"])){
  echo createService("go1920","/usr/local/go/bin/go run main.go","root",$BASE);
}


?>

<!--<script src="https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs/loader.js"></script>-->
<script src="https://unpkg.com/monaco-editor@latest/min/vs/loader.js"></script>
<script src="https://unpkg.com/monaco-go/dist/monaco.contribution.js"></script>


<script src="//hasandelibas.github.io/documenter/documenter.js?disable-html=true"></script>
<meta charset="utf8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<body class="show-menu theme-dark">
<header>
  <menu-opener icon-button onclick="$('body').classList.toggle('menu-hide')"><svg xmlns="http://www.w3.org/2000/svg" height="24" viewBox="0 -960 960 960" width="24"><path fill="currentColor" d="M120-240v-80h720v80H120Zm0-200v-80h720v80H120Zm0-200v-80h720v80H120Z"></path></svg></menu-opener>
  <div title>📓 GO - <?=$BASE?></div>
  <title>GO - <?=$BASE?></title>
  <div button onclick="
    let person = prompt('Please enter your name', 'Test');
    documenter.post('?new-folder='+person);
  ">New Folder</div>
  <div button onclick="
    let person = prompt('Please enter your name', 'Get');
    documenter.post('?new-file='+person);
  ">New File</div>

  <div button onclick="$('[api-modal]').style.display=null;">API TEST</div>

  <div class="space"></div>
  <!--
  USER-KEY:
  <input type="password" id="user-api-key">
  -->
<style>
  *{
    box-sizing:border-box;
  }
  body{
    --primary:#DE11BA;
    --primary:#AB11ED;
    font-family:"Ubuntu Mono", monospace!important;
  }
  [using]{
    height:2em;
    font-size:12px;
  }
  [using] a{
    color:#DE11BA;
    margin-right:1em;
  }
  h1,h2,h3{
    display:flex;
  }
  input,textarea,[response]{
    font-size:14px!important;
    border-radius:6px!important;
    background:#23241f!important;
    box-shadow:none!important;
  }
  api{
    position:relative;
    font-size:.8em;
    max-height:40vh;
    grid-template-rows: 1fr;
  }
  
  [button]{
    user-select:none;
    font-size:14px!important;
    border-radius:6px!important;
  }

  [copy]{
    position:absolute;
    right:.5em;
    top:.5em;
    z-index:99;
  }
  [response]{
    background: #8882;
    box-shadow: inset 0px 0px 0px 2px #8882;
    opacity: .8;
    padding: 0.5em;
    height: 100%;
    resize: none;
    margin:0;
    overflow:auto;
  }
  [password]{
    text-security: circle;
    -moz-text-security: circle;
    -webkit-text-security: circle;
  }
  .editor{
    width:100%;
    height:calc(100vh - 250px);
  }
</style>

<script>
  documenter.on("click","[code=curl]",function(){
    let root = this.parent.parent.parent
    let url  = "https://"+ location.hostname + $("[url]",root).value
    let body = $("[body]",root).value
    let responseDiv = $("[response]",root)
    const auth = $("#user-api-key").value;

responseDiv.innerHTML =   `curl $'${url}' \\
  -H 'Authorization: Bearer <span password> ${auth}</span> '  \\
  -H 'Content-Type: application/json' \\
  --data '${body}'
`
    
  });
  documenter.on("click","[request]",function(){
    let root = this.parent.parent.parent
    let url  = $("[url]",root).value
    let body = $("[body]",root).value
    let responseDiv = $("[response]",root)
    const auth = $("#user-api-key").value;

    console.log(auth);

    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + auth
      },
      body: body,
    })
    .then(res => res.text())
    .then(e=>{
      responseDiv.innerText = e
      responseDiv.innerHTML = JSON.stringify( JSON.parse(e), "\n", 2 )
    })
    .catch(console.error);
  });

    
  function CopyToClipboard(text) {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text).then(() => {
        console.log("Text copied to clipboard successfully!");
      }).catch(err => {
        console.error("Failed to copy text: ", err);
      });
    } else {
      // Fallback for older browsers
      const textarea = document.createElement("textarea");
      textarea.value = text;
      textarea.style.position = "fixed";  // Prevent scrolling to bottom
      document.body.appendChild(textarea);
      textarea.focus();
      textarea.select();

      try {
        document.execCommand("copy");
        console.log("Text copied using fallback method.");
      } catch (err) {
        console.error("Fallback copy failed: ", err);
      }

      document.body.removeChild(textarea);
    }
  }


  documenter.on("click","[copy]",function(){
    let code = documenter.render($("[response]",this.parent).outerHTML)
    $$("[password]",code).map(e=>e.removeAttribute('password'))
    
    CopyToClipboard(code.innerText)
    documenter.message("Copied!");
  })

  documenter.on("ready",function(){
    document.body.setAttribute("spellcheck", "false");
  })


  documenter.on("click","[button=test]",function(){
    let apiDiv = this.parent.next
    
    if(apiDiv.tagName == "API") {
      let codeDiv = this.parent.next.next 
      if(codeDiv.style.display=="none"){
        codeDiv.style.display=null
        apiDiv.style.display="none"
      }else{
        codeDiv.style.display="none"
        apiDiv.style.display=null
      }
      return
    }

    let codeDiv = this.parent.next 

    console.log(codeDiv.innerText)

    let code = codeDiv.innerText.split("\n")
    let urlCode = code.splice(0,1)[0]
    let bodyCode = code.join("\n").trim()
    
let html = `

<api grid-2 gap>
  <div flex-y gap>
    <input url value="${urlCode}">
    <textarea body style="height:auto;resize:none;">${bodyCode}</textarea>
    <div flex-x center gap>
      <div button code="curl">cURL</div>
      <space></space>
      <div button request>Run</div>
    </div>
  </div>
  <div copy button>Copy</div>
  <pre response></pre>
</api>


` 
    html = documenter.render(html)
    codeDiv.parent.insertBefore(html, codeDiv)

    setAutoHeight($("textarea",html))

    codeDiv.style.display="none"

  })

  function setAutoHeight(textarea) {
    // Reset the height so that shrinking works properly
    textarea.style.height = 'auto';
    
    // Set the height to match the scroll height
    textarea.style.height = `${textarea.scrollHeight}px`;
  }
</script>


<script>
documenter.on("ready",function(){
  $$("[part]").map(e=>{
    let val = e.getAttribute('part')
    if(e.hasAttribute("part-header")){
      $("menu").appendChild(documenter.render(`<a class="menu-h1" open='${val}' href='#${val}'>${val}</a>`))
    }else{
      $("menu").appendChild(documenter.render(`<a class="menu-h2" open='${val}' href='#${val}'>${val}</a>`))
    }
  })
  $$("menu a").find(e=>e.getAttribute("href")==location.hash)?.click()
})
let currentEditor = null
documenter.on("click","a",function(){
  if(currentEditor!=null) currentEditor._tempValue = currentEditor.getValue()
  let open = (new URL(this.href)).hash.substr(1)
  console.log(open)
  $$("[part]").map(e=>{
    if(e.getAttribute('part')==open){
      e.style.display=null
    }else{
      e.style.display="none"
    }
  })
  $$("menu a").map(e=>e.classList.remove("active"))
  this.classList.add("active")
  updateUsing(open)
  currentEditor = editors[open]
  //$("[part='"+this.getAttribute('open-part')+"']").style.display=null
})

function updateUsing(file){
  console.log(file)
  $("[part="+file+"] [using]").innerHTML = ""
  Object.entries(editors).map(_=>{
    let key = _[0]
    let editor = _[1]
    let val = editor.getValue()
    if( key!=file &&  val.includes(file)){
      let el = $("[part="+file+"]")

      $("[part="+file+"] [using]").innerHTML +="<a href='#"+key+"'>"+key+"</a>"
    }
  })
}
</script>

<script>
function htmlDecode(input) {
  var doc = new DOMParser().parseFromString(input, "text/html");
  return doc.documentElement.textContent;
}
//require.config({ paths: { 'vs': 'https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs' }});
require.config({ paths: { 'vs': 'https://unpkg.com/monaco-editor@latest/min/vs' } });






const editors = {}
let refreshDebounce = documenter.debounce(()=>{
  $$("menu a").find(e=>e.getAttribute("href")==location.hash)?.click()
},300)
documenter.when(".editor",function(el){
  require(["vs/editor/editor.main"], function () {
    if(el._value==null){ 
      el._value = htmlDecode(el.innerHTML)
      el.innerHTML = ""
    }else{
      return
    }
    
    let editor = monaco.editor.create(el, {
      value: el._value,
      language: "go",
      theme: "vs-dark", 
      automaticLayout: true,
      scrollBeyondLastLine: false,
      tabSize: 2,
    });
    editors[el.getAttribute("name")] = editor
    editor._tempValue = el._value
    editor.name = el.getAttribute('name')
    refreshDebounce()
  });
})


function save(){
  let file    = location.hash.substr(1)
  let title   = $("[part="+file+"]").getAttribute("title")
  let comment = $("[part="+file+"] textarea").value
  let code    = editors[file].getValue()
  documenter.post("?save=1",{
    file : title +"/"+ file,
    comment : comment,
    code : code 
  })
  documenter.loading()
}


function build(){
  documenter.post("?build=1",{}).then(e=>e.text()).then(e=>{
    if(e.trim()){ 
      documenter.info(e.trim())
    }else{
      documenter.info("✅ Success")
    }
  })
  documenter.loading()
}

function test(){
  documenter.post("?test=1",{}).then(e=>e.text()).then(e=>{
    if(e.trim()) documenter.info(e.trim())
  })
}

document.addEventListener('keydown', e => {
  if (e.ctrlKey && e.key === 's') {
    e.preventDefault();
    save()
  }
  if (e.ctrlKey && e.key === 'b') {
    e.preventDefault();
    build()
  }
})

</script>

<script>
  documenter.on("click","[create-from-sql]",function(){
    let sql = $("[description]",this.parent.parent.parent).value
    let name = this.parent.parent.parent.getAttribute("part")
    console.log(sql)
    documenter.post("?model=1",{
      data:sql,
      name:name
    }).then(e=>e.text()).then(e=>{
      editors[name].setValue(e)
    })
  })
</script>


<script>
// GO LANG HELPER





function AllLines(){
  return Object.values(editors).map(e=>{
    if(e==currentEditor){
      return e.getValue()
    }
    return e._tempValue
  }).join("\n")
}

function detectStructFields(){
  let lines = AllLines().split("\n");
  let functions = [];
  lines.forEach((line) => {
    let match = line.match(/type\s+([A-Za-z_][A-Za-z0-9_]*)\s+struct\s*/);
    if (match) {
      functions.push(match[1]);
    }
  });
  return functions;
}

function detectGoFunction(text) {
  let lines = AllLines().split("\n");
  let functions = [];
  lines.forEach((line) => {
    let match = line.match(/func\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(/);
    if (match) {
      functions.push(match[1]);
    }
  });
  return functions;
}


function detectAllGoFunctionParameters(text) {
  let lines = AllLines().split("\n");
  let functions = {};
  lines.forEach((line) => {
    let match = line.match(/func\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(/);
    if (match) {
      
      functions[match[1]] = [];
      let param =  line.match( /func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)/ );
      if (param) {
        functions[match[1]] = [...param[2].split(",").map(e=>e.trim().split(/\s+/).join(":") )];
      }
    }
  });
  return functions;
}


function detectGoCurrentFunctionParameters(text, index) {
  let lines = text.split("\n");
  let parameters = [];
  let functionFound = false;

  for ( let i = index; i >= 0; i-- ) {
    let line = lines[i];
    let tabIndent = line.match(/^\t*/)[0].length;
    let spaceIndent = line.match(/^ */)[0].length;
    let indent = tabIndent + spaceIndent/2;
    

    let match =  line.match( /func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)/ );
    if (match) {
      parameters.push(...match[2].split(",").map(e=>e.trim().split(" ")[0] ));
      break
    }

    if( indent==0 && line.trim()!="" ) break;
    

  }
  console.log(parameters);
  parameters = [...new Set(parameters)];
  return parameters;
}

function detectPythonVariables(text) {
  let lines = text.split("\n");
  let variables = [];
  lines.forEach((line) => {
    let match = line.match(/([a-zA-Z0-9_]*)\s*\:\=/);
    if (match) {
      variables.push(match[1]);
    }
    match = line.match(/var ([a-zA-Z0-9_]*)\s*/);
    if (match) {
      variables.push(match[1]);
    }
  });
  // unique variables
  variables = [...new Set(variables)];
  return variables;
}




require(['vs/editor/editor.main'], function () {




  function createDependencyProposals(range, text) {

    let staticFunctions = []

    let structFields = detectStructFields()
    let functions = detectGoFunction(text)
    let variables = detectPythonVariables(text)
    let parameters = detectGoCurrentFunctionParameters(text,range.startLineNumber-1)

    functions = functions.map(e => ({
      label: e,
      insertText: e , // +"()",
      range: range,
      kind: monaco.languages.CompletionItemKind.Function,
      sortText : "3",
      //detail:"Standart giriÅŸ Ã§Ä±kÄ±ÅŸ fonksiyonlarÄ±",
      //unit:"byte"
    }))

    variables = variables.map(e => ({
      label: e,
      insertText: e,
      range: range,
      kind: monaco.languages.CompletionItemKind.Variable,
      sortText : "2",
      //detail:"Standart giriÅŸ Ã§Ä±kÄ±ÅŸ fonksiyonlarÄ±",
      //unit:"byte"
    }))


    parameters = parameters.map(e => ({
      label: e,
      insertText: e,
      range: range,
      kind: monaco.languages.CompletionItemKind.Variable,
      sortText : "1",
      //detail:"Standart giriÅŸ Ã§Ä±kÄ±ÅŸ fonksiyonlarÄ±",
      //unit:"byte"
    }))



    structFields = structFields.map((e) => ({
      label: e,
      insertText: e,
      range: range,
      kind: monaco.languages.CompletionItemKind.Field,
      sortText: "4",
    }));


    let keywords = [
      "int","string","double",
      "if","else","for","return","func","var","const","struct","interface",
      "switch","case","defer","go","map","chan","package","import"] //,"import","from","as","class","try","except","finally","with","assert","global","nonlocal","lambda","del","yield","in","is","not","and","or","as","True","False","None"]
    keywords = keywords.map(e=>({
      label:e,
      insertText:e,
      range:range,
      kind: monaco.languages.CompletionItemKind.Keyword,
      documentaion:"Standart anahtar kelimeler",
      sortText:"9"
      //detail:"Standart giriÅŸ Ã§Ä±kÄ±ÅŸ fonksiyonlarÄ±",
      //unit:"byte"
    }))




    return [...functions, ...variables, ...parameters,...structFields,...keywords]

  }


    

  // 1️⃣ Detect all structs and their fields
  function detectGoStructs(text) {
    const lines = AllLines().split("\n");
    const structs = {};
    let currentStruct = null;

    for (let line of lines) {
      let structMatch = line.match(/type\s+([A-Za-z_][A-Za-z0-9_]*)\s+struct\s*{/);
      if (structMatch) {
        currentStruct = structMatch[1];
        structs[currentStruct] = [];
        continue;
      }

      if (currentStruct) {
        let fieldMatch = line.match(/^\s*([A-Za-z_][A-Za-z0-9_]*)\s+[A-Za-z0-9_\[\]*]+/);
        if (fieldMatch) structs[currentStruct].push(fieldMatch[1]);
        if (line.includes("}")) currentStruct = null;
      }
    }

    return structs; // { StructName: [field1, field2, ...] }
  }

  // 2️⃣ Detect variables and their types
  function detectGoVariableTypes(text) {
    const lines = text.split("\n");
    const variableTypes = {};

    lines.forEach(line => {
      //!!DEBUGGER

      // func( xxx type)
      let funcMatch =  line.match( /func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)/ );
      if (funcMatch) {
        funcMatch[2].split(",").map(e=>e.trim().split(/\s+/).join(" ")).map(e=>{
          let name = e.split(" ")[0], type = e.split(" ")[1].split("&").join("").split("*").join("")
          variableTypes[name] = type
        })
      }


      // var x Type
      let varMatch = line.match(/var\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+\**([A-Za-z_][A-Za-z0-9_]*)/);
      if (varMatch) {
        variableTypes[varMatch[1]] = varMatch[2];
        return;
      }

      // short declaration: x := Type{}
      let shortMatch = line.match(/([a-zA-Z_][a-zA-Z0-9_]*)\s*:=\s*([A-Za-z_][A-Za-z0-9_]*)\s*{}/);
      if (shortMatch) {
        variableTypes[shortMatch[1]] = shortMatch[2];
        return;
      }

      let shortMatchFn = line.match(/([a-zA-Z_][a-zA-Z0-9_]*)\s*:=\s*([A-Za-z_][A-Za-z0-9_]*)/);
      if (shortMatchFn) {
        if(shortMatchFn[2].startsWith("Model_")) shortMatchFn[2] = shortMatchFn[2].replace("Model_","Struct_Model_")
        if(shortMatchFn[2].startsWith("Func_")) shortMatchFn[2] = shortMatchFn[2].replace("Func_","Struct_Model_")
        if(shortMatchFn[2].startsWith("Get_")) shortMatchFn[2] = shortMatchFn[2].replace("Get_","Struct_Get_")
        variableTypes[shortMatchFn[1]] = shortMatchFn[2];
        return;
      }
    });

    return variableTypes; // { user: "User", u: "User" }
  }

  // 3️⃣ Get fields for a specific object
  function detectStructFieldsForObject(text, objectName) {
    const structs = detectGoStructs(text);
    const variableTypes = detectGoVariableTypes(text);

    const structName = variableTypes[objectName];
    if (structName && structs[structName]) {
      return structs[structName].map(field => ({
        label: field,
        insertText: field,
        kind: monaco.languages.CompletionItemKind.Field,
        sortText: "4"
      }));
    }

    return [];
  }



  monaco.languages.registerCompletionItemProvider('go', {
    triggerCharacters: ['.'],
    provideCompletionItems: function (model, position) {


      const lineContent = model.getLineContent(position.lineNumber);
      const objectMatch = lineContent.slice(0, position.column - 1).match(/([a-zA-Z_][a-zA-Z0-9_]*)\.[a-zA-Z0-9_]*$/);

      if (objectMatch) {
        const objectName = objectMatch[1];
        const suggestions = detectStructFieldsForObject(model.getValue(), objectName);
        return { suggestions };
      }
      
      var word = model.getWordUntilPosition(position);
      var range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn
      };
      return {
        suggestions: createDependencyProposals(range, model.getValue())
      };
    },
  })



  monaco.languages.registerSignatureHelpProvider('go', {
    signatureHelpTriggerCharacters: ['(', ','],
    provideSignatureHelp: function (model, position) {
      const word = model.getWordUntilPosition(position);

      const lineContent = model.getLineContent(position.lineNumber);
      
      const textBeforeCursor = lineContent.substring(0, position.column - 1);
      const commaCount = (textBeforeCursor.match(/,/g) || []).length;

      const fnParameters = detectAllGoFunctionParameters(AllLines());
      //console.log(fnParameters)
      console.log(lineContent)
      for(let fn in fnParameters){
        let params = fnParameters[fn]
        console.log(lineContent)
        let founded = lineContent.split(/\W/).includes(fn)
        if (founded) {
          return {
            value: {
              signatures: [
                {
                  documentation: fn + '('+params.map(e=>e.split(":")[0]).join(", ")+')',
                  label: params.map(e=>e.split(":").join(": ")).join(", "),
                  parameters:  params.map(e=>{
                    return {
                      label: e.split(":").join(": "),
                      documentation: e.split(":").join(": "),
                    }
                  }),
                }
              ],
              activeSignature: 0,
              activeParameter: commaCount
            },
            dispose: () => {}
          };
        }
      }


      return { value: { signatures: [], activeSignature: 0, activeParameter: 0 }, dispose: () => {} };
    }
  });
});
</script>


<div panel="bottom-right">
  <button onclick="updateTask()"> TASKS </button>
  <panel tasks style="width:240px;padding:0;">
    
  </panel>
</div>

<script>
function updateTask(){
  tasks = Object.entries(editors).map(key_editor=>{
    let key = key_editor[0]
    let editor = key_editor[1]
    let infos = editor.getValue().split("\n").map(e=>e.match(/\/\/\!(.*)/gm)).map(e=>e==null?[]:e).map((e,i)=>e.length>0?(i+1)+":"+e[0]:"").filter(e=>e)
    if(infos.length>0) return infos.map(e=>key + ":" + e)
    return [];
  }).flat()

  $("[tasks]").innerHTML = tasks.map(e=>{
    let file = e.split(":")[0]
    let index = e.split(":")[1]
    let comment = e.substr(file.length+2+index.toString().length+3).trim()
    return `<a hover padding flex-y open="${file}" style="color:var(--front);font-size:.8em;" href="#${file}"><div>${comment}</div><div style='opacity:.8;font-size:.8em;'>${file} : ${index}</div></a>`

  }).join("")

  
  return tasks

}
</script>

</header>


<content>
  <menu></menu>
  <main>

<?php 


$title = "";

$src   = $BASE . "Source/*/*.go";
$files = glob($src);
sort($files);
/*
print_r($src);
print_r($files);
exit();
*/
foreach ($files as $filename) {
  $name = str_replace($BASE."Source/","",$filename);
  $name = str_replace(".go","",$name);
  $_title = explode("/",$name)[0];
  if($title!=$_title){
    echo "<div part='".$_title."' part-header><h1> " . $_title . "</h1></div>";
    $title = $_title;
  }
  $_name = explode("/",$name)[1];

  $source = file_get_contents($filename);
  $comments = comments($source);
  $description = @$comments[0];
  

  $code = $source;
  foreach($comments as $comment){
    $code = str_replace("/*".$comment."*/","",$code);
  }

  if(!$description) $description = "";
  if(!$code) $code = "";

  echo "\n";
  echo "<div title='".$title."' part='".$_name."'>\n";
  echo "  <h2 flex-sx center gap>" . $_name . "<div><div button style='font-size:.5em' create-from-sql>Create From Sql</div></div></h2>\n";
  echo "  <div using></div>\n";
  echo "  <textarea description style='width:100%;height:7em;margin-bottom:.5em;resize:vertical;'>". str_replace("<","&lt;",$description) . "</textarea>\n";
  echo "  <div class='editor' name='".$_name."' style='border-radius:6px;'>" . str_replace("<","&lt;",$code) ."</div>\n";
  
  echo "</div>";
  /*
  ```
  /!api/role/add/[GROUP_NICK]
  {
    "name"         : "Role Name",
    "color"        : "#DE11BA",
    "power_group"  : 0,
    "power_role"   : 0,
    "power_channel": 0,
    "power_post"   : 0,
    "power_member" : 0,
    "power_room"   : 0,
    "power_team"   : 0
  }
  ``` 
  */
}


?>
<div api-modal style='display:none;
  position: fixed;
  left: 0;
  top: 50px;
  width: 100%;
  height: calc(100% - 50px);
  background: var(--back);
  z-index: 99;
  padding: 1em;
  box-sizing: border-box;'>
  <h1 flex-sx center gap>
    API TEST
    <button onclick="$('[api-modal]').style.display='none'" >Close</button>
  </h1>
  
  <api grid-2 gap>
  <div flex-y gap>
    <input url value="/!apis">
    <textarea body style="height: 243px; resize: none;">[{
  "api": "/!api/test/ip"
},{
  "api": "/!api/user/id"
},{
  "api": "/!api/group/get",
  "data": 1
}]</textarea>
    <div flex-x center gap>
      <div button code="curl">cURL</div>
      <space></space>
      <div button request>Run</div>
    </div>
  </div>
  <div copy button>Copy</div>
  <pre response></pre>
</api>

</div>
</main>
</content>