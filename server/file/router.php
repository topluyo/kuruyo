<?php

$request = $_SERVER['REQUEST_URI'];

if (isset($_SERVER['HTTP_X_FORWARDED_PROTO'])) {
  $_SERVER['REQUEST_SCHEME'] = $_SERVER['HTTP_X_FORWARDED_PROTO'];
} else {
  $_SERVER['REQUEST_SCHEME'] = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off') ? 'https' : 'http';
}



// Serve the file directly if it exists
if (php_sapi_name() === 'cli-server') {
    $path = parse_url($_SERVER["REQUEST_URI"], PHP_URL_PATH);
    $fullPath = __DIR__ . $path;

    if (is_file($fullPath)) {
        return false;
    }
}

// Custom routing logic
$requestUri = trim(parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH), '/');

if ($requestUri === 'folder') {
    $_GET['page'] = 'file-manager';
    $_GET['path'] = '/';
} elseif (preg_match('#^folder/(.*)$#', $requestUri, $matches)) {
    $_GET['page'] = 'file-manager';
    $_GET['path'] = '/' . $matches[1];
} elseif (preg_match('#^pages/(.*)$#', $requestUri, $matches)) {
    $_GET['page'] = $matches[1];
}

// Correct WAY
if (preg_match('#^/folder/(.*)$#', $requestUri, $matches)) {
    $_GET['page'] = 'file-manager';
    $_GET['path'] = '/' . $matches[1];
}



# .dot dosyalarını engelle
if (preg_match('/\/\./', $request)) {
  http_response_code(403);
  die("<html lang='en'><head><title>403 - Forbidden</title></head><body style='user-select:none;background:#CCC;font-size:5vmin;display:flex;flex-direction:column;align-items:center;justify-content:center;font-family:system-ui;height: 100%;padding: 0;margin: 0;'><div>Forbidden</div><div style='font-size:.5em;opacity:.6'>403</div></body></html>");
}

if(is_file($path."/index.php")){
  require $path . "/index.php";
}

if(is_file($path."/index.html")){
  require $path . "/index.html";
}

// Route everything to index.php
require __DIR__ . '/index.php';
