<?php

require_once __DIR__."/engine.php";

FileManager::DefineOptions([
  "BACK_END_FILE_MANAGER_ROOT" => "/",
  "CREATE_THUMBNAIL" => false,
  "THUMBNAIL_WIDTH" => [60],
  "SHOW_FOLDER_SIZE" => false,
  "SECURITY_ENABLED" => true,
  "SECURITY_USERS" => [ "b0zkaşlsd1ıa0jkmzxkc081u2njlej1n0djuasd" => "b0zkaşlsd1ıa0jkmzxkc081u2njlej1n0djuasd" ],
  "LOGIN_HASH" => "ROOT-RANDOM-HASH-14axqaszczxc",
]);


// If running directy
if( FileManager::endsWith( $_SERVER['SCRIPT_FILENAME'], __FILE__) or FileManager::endsWith( __FILE__, $_SERVER['SCRIPT_FILENAME']) ){ 
  FileManager::Run_All();
}
