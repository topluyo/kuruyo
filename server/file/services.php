<?php 


require_once __DIR__."/index.php";

FileManager::Run_Security();



function createService($name, $exec,$desciption=null, $workdir = null,$user = null) {
  if ($user === null) {
      $user = get_current_user();
  }
  if ($workdir === null) {
    $workdir = getcwd();
  }
  if ($desciption === null) {
    $desciption = $name . " service";
  }

  $service = <<<SERVICE
[Unit]
Description=$desciption
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


if($_GET["create-service"]==1){
  echo createService($_POST['name'],$_POST['exec'],$_POST['description'],$_POST['workdir']);
  
}


if($_GET['restart']){
  shell_exec("systemctl restart ".$_GET['restart']);
  echo $_GET['restart'] . " restarted";
}

?>

<script src="//hasandelibas.github.io/documenter/documenter.js"></script>
<meta charset="utf8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<body>
<header body-class="show-menu theme-dark">
  <div title="">⚙️ Services</div>
  <div class="space"></div>
</header>



# Services

```
/etc/systemd/system/*
```

<?php 

$services = shell_exec("systemctl list-unit-files --type=service --all --no-legend | awk '{print $1}'");
$services = explode("\n",$services);
foreach($services as $s){
  if(preg_match("/^(go|node|php|sh)\d+\.service/", $s)){
    $name = str_replace(".service","",$s);
    echo "## $s <a href='?restart=$s'>Restart</a> <a target='_blank' href='https://terminal.uyguluyo.com/?run=journalctl -u $name -f'>Journal</a>\n";
    echo "```\n" . file_get_contents("/etc/systemd/system/$s") . "\n```\n";
  }
}
?>


# New Service
<form grid-form method="post" action="?create-service=1">
  <label>Examples</label>

  <div>
```
/usr/local/go/bin/go
/usr/bin/php
/usr/bin/node
/usr/bin/sh
```
  </div>

  <label>Name:</label>
  <input name="name">

  <label>Exec:</label>
  <input name="exec">

  <label>Description:</label>
  <input name="description">


  <label>WorkDir:</label>
  <input name="workdir" value="/var/web/services.topluyo.com/">


  <label></label>
  <button>Create</button>
</form>
