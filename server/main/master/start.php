<?php

require_once __DIR__."/engine.php";

FileManager::DefineOptions([
  "MASTER_URL" => "/master/",
  "BACK_END_FILE_MANAGER_ROOT" => "/",
  "CREATE_THUMBNAIL" => false,
  "THUMBNAIL_WIDTH" => [60],
  "SHOW_FOLDER_SIZE" => false,
  "SECURITY_ENABLED" => true,
  "SECURITY_USERS" => [ "fg76thgy8ghhuyuhkjghfı766hvg" => "5a9çöbxrı*kmmnıuyes634y*9gddGFJGGTrtugyvy" ],
  "LOGIN_HASH" => "ROOT-RANDOM-HASH-14axqaszczxc",
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
