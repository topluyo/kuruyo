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


}else{
  return require "index.html";
}


//echo $_SERVER["HTTP_HOST_PATH"] | "";