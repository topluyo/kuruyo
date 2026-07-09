<?php 

$PAGE = "api";
$BASE = trim(parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH), '/');
$BASE = $_SERVER['REQUEST_URI'];
$_prefix = ($_SERVER["HTTP_HOST_PATH"] | "") . "/editor/root/";
$_prefix = str_replace("//","/",$_prefix);


if (str_starts_with($BASE, $_prefix)) {
  $BASE = substr($BASE, strlen($_prefix));
  $PAGE = "main";
}


if($PAGE=="api"){
  if(@$_POST["tree"]){
    $path = $_POST["tree"];
    $result = [];
    $iterator = new RecursiveIteratorIterator(
      new RecursiveDirectoryIterator($path, FilesystemIterator::SKIP_DOTS),
      RecursiveIteratorIterator::SELF_FIRST
    );
    foreach ($iterator as $file) {
      $result[] = [
        'path' => $file->getPathname(),
        'type' => $file->isDir() ? 'dir' : 'file'
      ];
    }
    echo json_encode($result, JSON_PRETTY_PRINT);
  }

  if(@$_POST["cat"]){
    $path = $_POST["cat"];
    header('Content-Type: text/plain; charset=UTF-8');
    echo file_get_contents($path);
  }
  if(@$_POST["save"]){
    $path = $_POST["save"];
    echo file_put_contents($_POST["save"], base64_decode($_POST["data"]));
  }

  if(@$_POST["mkdir"]){
    $path = $_POST["mkdir"];
    header('Content-Type: text/plain; charset=UTF-8');
    if (!is_dir($path) && !mkdir($path, 0775, true)) {
      exit('"error"');
    }
    echo "success";
  }

  
  if(@$_POST["rm"]){
    $path = $_POST["rm"];
    header('Content-Type: text/plain; charset=UTF-8');
    exec("rm -rf \"".addslashes($path)."\"");
    echo "success";
  }

  if(@$_POST["touch"]){
    $path = $_POST["touch"];
    header('Content-Type: text/plain; charset=UTF-8');
    exec("touch \"".addslashes($path)."\"");
    echo "success";
  }
}else{
  return require "index.html";
}


//echo $_SERVER["HTTP_HOST_PATH"] | "";
