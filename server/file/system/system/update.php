<?php 
header("Content-Type: text/plain");
echo "starting updating...\n";


$domain = $_SERVER['SERVER_NAME'];
if(substr($domain,0,4)=="www.") $domain = substr($domain,4);
  

if($domain=="apps.asenax.com" || $domain=="master.asenax.com") {
  echo "This is master!";
  exit();
}


echo "downloading https://apps.asenax.com/system/system/get-updates.php\n";

$data = file_get_contents("https://apps.asenax.com/system/system/get-updates.php");
$data =  json_decode( $data, true );

if($data){ 
  echo "downloaded  https://apps.asenax.com/system/system/get-updates.php\n"; 
}

echo "==========================================================\n";
foreach($data as $path=>$code){
  file_put_contents(__DIR__."/".$path,$code);  
  echo "📄 $path \n"; 
}

?>
UPDATED ✅