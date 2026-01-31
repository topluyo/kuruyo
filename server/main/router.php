<?php

error_reporting(E_ALL);
ini_set('display_errors', 1);


$request = $_SERVER['REQUEST_URI'];

if (isset($_SERVER['HTTP_X_FORWARDED_PROTO'])) {
  $_SERVER['REQUEST_SCHEME'] = $_SERVER['HTTP_X_FORWARDED_PROTO'];
} else {
  $_SERVER['REQUEST_SCHEME'] = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off') ? 'https' : 'http';
}


# .dot dosyalarını engelle
if (preg_match('/\/\./', $request)) {
  http_response_code(403);
  die("<html lang='en'><head><title>403 - Forbidden</title></head><body style='user-select:none;background:#CCC;font-size:5vmin;display:flex;flex-direction:column;align-items:center;justify-content:center;font-family:system-ui;height: 100%;padding: 0;margin: 0;'><div>Forbidden</div><div style='font-size:.5em;opacity:.6'>403</div></body></html>");
}

# Dosya varsa normal sun
$path = __DIR__ . $request;
if (is_file($path)) {
  return false; // PHP built-in server dosyayı direkt servis etsin
}

if(is_file($path."/index.php")){
  return require $path . "/index.php";
}

if(is_file($path."/index.html")){
  return require $path . "/index.html";
}

$list=explode("/", preg_replace('/^\//', '', explode("?",$request)[0]));
for($i=0;$i<count($list);$i+=0){
  $f = __DIR__."/".implode("/",$list)."/index.php";
  if(is_file($f)){
    return require $f;
  }
  array_pop($list);
}
# Geri kalanı index.php'ye yönlendir
require __DIR__ . "/index.php";