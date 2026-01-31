<?php 

ob_start();
include(__DIR__."/make.php");
ob_end_clean();

header('Content-Type: application/json; charset=utf-8');

echo  json_encode([
  "lib/db.php" => file_get_contents(__DIR__."/lib/db.php"),
  "lib/df.php" => file_get_contents(__DIR__."/lib/df.php"),
  "lib/functions.php" => file_get_contents(__DIR__."/lib/functions.php"),
  "lib/request.php" => file_get_contents(__DIR__."/lib/request.php"),
  ".htaccess"  => file_get_contents(__DIR__."/.htaccess"),
  "index.php"  => file_get_contents(__DIR__."/index.php"),
]);

?>