<?php

require_once __DIR__."/engine.php";

FileManager::DefineOptions([
  "MASTER_URL" => "/explorer/",
  "BACK_END_FILE_MANAGER_ROOT" => "/",
  "CREATE_THUMBNAIL" => false,
  "THUMBNAIL_WIDTH" => [60],
  "SHOW_FOLDER_SIZE" => false,
  "SECURITY_ENABLED" => true,
  "SECURITY_USERS" => [ "XXXXXXXXXXXXXXXXX" => "XXXXXXXXXXXXXXXXXX" ],
  "LOGIN_HASH" => "XXXXXXXXXXXXXXX",
]);

function FileManagerStart(){
  $_GET["base"] = FileManager::$MASTER_URL;
  $_GET["path"] = "/";

  $requestUri = trim(parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH), '/');
  if(FileManager::startsWith($requestUri, substr(FileManager::$MASTER_URL,1) . "root")){
    $_GET['page'] = 'file-manager';
    $_GET["path"] = substr($requestUri,strlen( substr(FileManager::$MASTER_URL,1) . "root"));
  }
  FileManager::Run_All();
}
