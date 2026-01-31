<?php

ini_set('display_errors', 1); 
ini_set('display_startup_errors', 1); 
error_reporting(E_ALL);



require_once __DIR__."/system/lib/df.php";
require_once __DIR__."/system/lib/request.php";
require_once __DIR__."/system/lib/functions.php";
require_once __DIR__."/system/lib/db.php";


$db = new DB("localhost", "db","root","");

