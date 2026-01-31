<?php


function router($route,$prefix="") {
  $uri = $_SERVER['REQUEST_URI'];
  $uri = strtok($uri, '?');
  $uri = preg_replace('/\/+/', '/', $uri);        // remove double ///
  $uri = preg_replace('/\/$/', '' , $uri);        // remove last /

  if($route == "") $route="/";
  $route = preg_replace('/\/+/', '/', $route);    // remove double ///
  $route = preg_replace('/\/$/', '' , $route);    // remove last /
  $route = str_replace('/', '\/', $route);        // convert to RegExp string
  $route = preg_replace('/:([\w\_]+)/', '(?P<$1>[^\/]+)', $route);
  $route = '/^' . $route . '$/';

  if(( $route=="/^\\$/" && $uri=="" )){ // root
    return true;
  }

  if (preg_match($route, $uri, $matches)   ) {
    $params = [];
    foreach ($matches as $key => $value) {
      if (is_string($key)) {
        $params[$key] = urldecode($value);
      }
    }
    foreach ($params as $key => $value) {
      $GLOBALS[$prefix . $key] = $value;
      $_GET[$prefix . $key] = $value;
      $_REQUEST[$prefix . $key] = $value;
    }
    if($params) return $params;
    return true;
  } else {
    return false;
  }
}


function domain(){
  $domain = $_SERVER['SERVER_NAME'];
  if(substr($domain,0,4)=="www.") $domain = substr($domain,4);
  return $domain;
}

function url($path=null){
  if($path==null){
    return trim($_SERVER["SCRIPT_URL"] , "/");
  }

  $domain = $_SERVER['SERVER_NAME'];
  if(substr($domain,0,4)=="www.") $domain = substr($domain,4);

  if ( $_SERVER["SERVER_PORT"]==443 ){
    $protocol = 'https://';
  } else {
    $protocol = 'http://';
  }
  $path = trim($path);
  if($path=="") $path = "/";
  if($path[0]=="/"){
    $url =  $domain . "/" . $path;

    $url = preg_replace('/\/+/', '/', $url);    // remove double ///
    $url = preg_replace('/\/$/', '' , $url);    // remove last /
    return $protocol . $url;
  }else{
    $base = $_SERVER["REQUEST_URI"];
    $base = explode("/",$base);
    array_pop($base);
    $base = join("/", $base);
    $url = $domain . "/" . $base . "/" . $path ;

    $url = preg_replace('/\/+/', '/', $url);    // remove double ///
    $url = preg_replace('/\/$/', '' , $url);    // remove last /
    return $protocol . $url;
  }
}

function path(...$args) {
  $isFirstDelimiter = trim($args[0])[0]=="/" ;
  $path = join('/', $args);
  $path = preg_replace('#/+#','/', $path);
  $parts = explode('/', $path);
  $finalParts = [];
  foreach ($parts as $part) {
    if ($part === '..') {
      array_pop($finalParts);
    }elseif ($part !== '' && $part !== '.') {
      $finalParts[] = $part;
    }
  }
  $fixedPath = implode('/', $finalParts);
  if($isFirstDelimiter) $fixedPath = "/" . $fixedPath;
  return $fixedPath;
}


function upload($source,$folder,$name=false){
  $FILE = $_FILES[$source];
  function extension($s) {$n = strrpos($s,"."); if($n===false) return "";return substr($s,$n+1);};
  if($name==false){
    $bytes = random_bytes(32);
    $name = bin2hex($bytes) . "." . extension( $FILE["name"] );
  }
  $target = path($folder,$name);
  write($target);
  if ($FILE["error"] == UPLOAD_ERR_OK) {
    $result = move_uploaded_file($FILE["tmp_name"], $target);
    return $result ? $name : false;
  }
  return false;
}

function post($param=null,$default=false){
  // If request method is post
  if($param==null){
    if($_SERVER['REQUEST_METHOD']=='POST'){
      return true;
    }else{
      return false;
    }
  }
  if(isset($_POST[$param])){
    return $_POST[$param];
  }
  return $default;
}

function get($param,$default=false){
  // If request method is post
  if($param==null){
    if($_SERVER['REQUEST_METHOD']=='GET'){
      return true;
    }else{
      return false;
    }
  }
  if(isset($_GET[$param])){
    return $_GET[$param];
  }
  return $default;
}

function all($param,$default=false){
  if(isset($_REQUEST[$param])){
    return $_REQUEST[$param];
  }
  return $default;
}

function alls($array,$func=null){
  $response = [];
  if(is_string($array)){
    $array = array_map("trim",explode(",",$array)) ;
  }
  foreach($array as $key=>$value){
    if(is_int($key)){
      $response[$value] = all($value);
    }else{
      $response[$key] = all($key,$value);
    }
  }
  if(is_callable($func)){
    foreach($response as $key=>$value){
      $response[$key] = call_user_func($func,$value);
    }
  }
  if(is_array($func)){
    $response=array_merge($response,$func);
  }
  return $response;
}

function json($data=null){
  if($data!==null){
    header('Content-Type: application/json');
    echo json_encode($data);
    exit;
  }else{
    return json_decode(file_get_contents('php://input'),true);
  }
}

function success($data="",$message=""){
  json( ["status"=>"success","data"=>$data, "message"=>$message] );
}

function error($data="",$message=""){
  json( ["status"=>"error","data"=>$data, "message"=>$message] );
}
/*
function session($param,$default=false){
  // If session started
  if(session_status()==PHP_SESSION_ACTIVE){
    if(isset($_SESSION[$param])){
      return $_SESSION[$param];
    }
  }
  return $default;
}
*/

function session(string $param, $value = null) {
  if (session_status() !== PHP_SESSION_ACTIVE) {
    session_start();
  }
  if ($value === null) {
    return $_SESSION[$param] ?? null;
  }
  if ($value === ''||$value === false) {
    unset($_SESSION[$param]);
  } else {
    $_SESSION[$param] = $value;
  }
}

function cookie($param, $value=null, $expire=null){
  if($value!==null){
    if($expire!==null){
      setcookie($param, $value, $expire);
    }else{
      setcookie($param, $value );
    }
  }else{
    if(isset($_COOKIE[$param]) && $_COOKIE[$param]!="" && $_COOKIE[$param]!==null){
      return $_COOKIE[$param];
    }
  }
  return false;
}



function message($text=null){
  @session_start();
  if($text==null){
    $message =  isset( $_SESSION["_message_"] ) ? $_SESSION["_message_"] : "" ;
    if(isset($_SESSION["_message_"])){
      unset($_SESSION["_message_"]);
    }
    return $message;
  }else{
    $_SESSION["_message_"] = $text;
  }
}

function redirect($url=null, $permanent = false){
  if($url==null){ $url = $_SERVER['REQUEST_URI']; }
  header('Location: ' . $url, true, $permanent ? 301 : 302);
  exit();
}




$__LOGIN_HASH__ = 'RAND0M-T€XT-F0R-5€5510N'.$_SERVER['DOCUMENT_ROOT']; // Random Text
$__LOGIN_NAME__ = strtolower(trim(preg_replace('/[^A-Za-z0-9-]+/', '-', $_SERVER['SERVER_NAME'].'-master-session' )));
$__LOGIN_TIME__ = 3600 * 24 * 360; // 1 Year

function login($id, $__login_name__ = null){
  global $__LOGIN_NAME__,$__LOGIN_HASH__,$__LOGIN_TIME__;
  if($__login_name__==null) $__login_name__ = $__LOGIN_NAME__;

  if(strpos($id,"___")!==false) {
    echo "ID should not contain '___'";
    exit();
  }

  $hash = [
    $id,
    md5($__LOGIN_HASH__.md5($id.$__LOGIN_HASH__)),
    time() + $__LOGIN_TIME__,
    md5($__LOGIN_HASH__.md5( time() + $__LOGIN_TIME__ . $__LOGIN_HASH__ ))
  ];
  setcookie( $__login_name__   ,json_encode($hash), time() + $__LOGIN_TIME__ , "/" );
}

function logout($__login_name__ = null){
  global $__LOGIN_NAME__,$__LOGIN_HASH__,$__LOGIN_TIME__;
  if($__login_name__==null) $__login_name__ = $__LOGIN_NAME__;

  unset($_COOKIE[$__login_name__]); 
  setcookie($__login_name__, '[]', -1 , "/" ); 
}

function isLogged($__login_name__= null){
  return id($__login_name__) === false ? false : true;
}

function id($__login_name__= null){
  global $__LOGIN_NAME__,$__LOGIN_HASH__,$__LOGIN_TIME__;
  if($__login_name__==null) $__login_name__ = $__LOGIN_NAME__;

  if( !isset($_COOKIE[$__login_name__]) ) return false;
  if( $_COOKIE[$__login_name__]=="" ) return false;
  if( $_COOKIE[$__login_name__]==null ) return false;
  
  
  $hash = [];
  $hash = @json_decode( $_COOKIE[$__login_name__] , false );
  if( count($hash) < 4 ) return false;

  $id          = $hash[0];
  $id_verify   = $hash[1];
  $time        = $hash[2];
  $time_verify = $hash[3];
  

  if( $id_verify != md5($__LOGIN_HASH__.md5($id.$__LOGIN_HASH__)) ) return false;
  if( $time_verify != md5($__LOGIN_HASH__.md5($time.$__LOGIN_HASH__)) ) return false;
  if(time() > $time) return false;

  return $id;
}



function blade($code) {
  $code = preg_replace("/{{\?(.+)}}/","<?= @isset($1) ? $1 : '' ?>",$code);
  $code = str_replace("@{{","<?php ",$code);
  $code = str_replace("{{","<?= ",$code);
  
  $code = str_replace("}}\n"," ?> \n",$code);
  $code = str_replace("}}"," ?>",$code);

  $code = preg_replace("/@foreach(.+)/","<?php foreach $1   {?>",$code);
  $code = preg_replace("/@for(.+)/","<?php for $1 {?>",$code);
  $code = preg_replace("/^\s*@else if(.+)/m","<?php } else if $1 { ?>",$code);
  $code = preg_replace("/^\s*@if(.+)/m","<?php if $1 {?>",$code);
  $code = preg_replace("/@if(.+)/","<?php if $1 {?>",$code);
  $code = preg_replace("/@elseif(.+)/","<?php } else if $1 {?>",$code);
  $code = preg_replace("/@else if(.+)/","<?php } else if $1 {?>",$code);
  $code = str_replace("@else","<?php } else {?>",$code);
  $code = preg_replace("/^\s*@end(.*)/m","<?php } ?>",$code);

  return $code;
}