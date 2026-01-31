<?php

//ini_set('display_errors', 1); ini_set('display_startup_errors', 1); error_reporting(E_ALL);


class FileManager{

  public static $MASTER_URL = "/";

  public static $BACK_END_FILE_MANAGER_ROOT="";

  public static $CREATE_THUMBNAIL = true;

  public static $THUMBNAIL_WIDTH = [60,1080];

  public static $THUMBNAIL_FORMAT = null;

  public static $SHOW_FOLDER_SIZE = false;

  public static $SECURITY_ENABLED = true;

  public static $SECURITY_USERS = [];

  public static $READ_ONLY = false;

  public static $EXTENSIONS = "*";


  //------------- SOURCE -----------------
  // ["/path/to/file"=>"file content"]
  public static $SOURCE = [];
  public static function file_get_contents($file){
    if(file_exists(__DIR__."/pages.php")){
      require_once __DIR__."/pages.php";
    }
    $sourceTarget = $file;
    if( substr($file, 0, strlen(__DIR__."/"))==__DIR__."/" )
      $sourceTarget = substr($file, strlen(__DIR__."/"));

    if(file_exists($file)){
      return file_get_contents($file);
    }else if( isset(self::$SOURCE[$sourceTarget]) ){
      return base64_decode( self::$SOURCE[$sourceTarget] );
    }else{
      return "FileManagerStarter:: '$sourceTarget' not found. Full path: $file";
    }
  }
  //============= SOURCE =================


  public static function path(...$args){
    $path = "";
    foreach($args as $arg){
      $path = rtrim($path, "/")."/".ltrim($arg, "/");
    }
    // If On Windows
    if( strtoupper(substr(PHP_OS, 0, 3)) === 'WIN' ){
      $path = str_replace("/", "\\", $path);
      if( substr($path, 0, 1)=="\\" ){
        $path = substr($path, 1);
      }
    }
    return $path;
  }

  public static function linuxPath($path){
    return str_replace("\\", "/", $path);
  }


  public static function ResponseJson($data){
    header('Content-Type: application/json');
    if( is_string($data) ){
      echo $data;
    }else{
      echo json_encode($data,JSON_INVALID_UTF8_IGNORE);
    }
    exit();
  }
  public static function ResponseSuccess($data){
    self::ResponseJson(["status"=>"success","data"=>$data]);
  }
  public static function ResponseError($data){
    self::ResponseJson(["status"=>"error","data"=>$data]);
  }

  public static function ResponseFile($file){
    $mime = mime_content_type($file);
    if(self::endsWith($file,".js")) $mime = 'application/javascript';
    header("Content-Type: $mime; charset=utf-8");
    echo file_get_contents($file);
    exit();
  }

  public static function ResponseImage($file){
    if (file_exists($file)) {
      if( self::endsWith($file,".svg") ){
        header('Content-Type: image/svg+xml');
      }else{
        $image_info = getimagesize($file);
        //Set the content-type header as appropriate
        header('Content-Type: ' . $image_info['mime']);
      }
      //Set the content-length header
      header('Content-Length: ' . filesize($file));
      //Write the image bytes to the client
      readfile($file);
      exit();
    }
  }

  public static function ResponseType($filename){
    header('Content-Type: '.self::GetMimeType($filename));
  }

  public static function GetMimeType($filename){
    $idx = explode( '.', $filename );
    $count_explode = count($idx);
    $idx = strtolower($idx[$count_explode-1]);

    $mimet = array( 
      'txt' => 'text/plain',
      'htm' => 'text/html',
      'html' => 'text/html',
      'php' => 'text/html',
      'css' => 'text/css',
      'js' => 'application/javascript',
      'json' => 'application/json',
      'xml' => 'application/xml',
      'swf' => 'application/x-shockwave-flash',
      'flv' => 'video/x-flv',

      // images
      'png' => 'image/png',
      'jpe' => 'image/jpeg',
      'jpeg' => 'image/jpeg',
      'jpg' => 'image/jpeg',
      'gif' => 'image/gif',
      'bmp' => 'image/bmp',
      'ico' => 'image/vnd.microsoft.icon',
      'tiff' => 'image/tiff',
      'tif' => 'image/tiff',
      'svg' => 'image/svg+xml',
      'svgz' => 'image/svg+xml',

      // archives
      'zip' => 'application/zip',
      'rar' => 'application/x-rar-compressed',
      'exe' => 'application/x-msdownload',
      'msi' => 'application/x-msdownload',
      'cab' => 'application/vnd.ms-cab-compressed',

      // audio/video
      'mp3' => 'audio/mpeg',
      'qt' => 'video/quicktime',
      'mov' => 'video/quicktime',

      // adobe
      'pdf' => 'application/pdf',
      'psd' => 'image/vnd.adobe.photoshop',
      'ai' => 'application/postscript',
      'eps' => 'application/postscript',
      'ps' => 'application/postscript',

      // ms office
      'doc' => 'application/msword',
      'rtf' => 'application/rtf',
      'xls' => 'application/vnd.ms-excel',
      'ppt' => 'application/vnd.ms-powerpoint',
      'docx' => 'application/msword',
      'xlsx' => 'application/vnd.ms-excel',
      'pptx' => 'application/vnd.ms-powerpoint',


      // open office
      'odt' => 'application/vnd.oasis.opendocument.text',
      'ods' => 'application/vnd.oasis.opendocument.spreadsheet',
    );

    if (isset( $mimet[$idx] )) {
      return $mimet[$idx];
    } else {
      return 'application/octet-stream';
    }
  }


  public static function BeatifulPath($path){
    $path = str_replace(array('/', '\\'), DIRECTORY_SEPARATOR, $path);
    $parts = array_filter(explode(DIRECTORY_SEPARATOR, $path), 'strlen');
    $absolutes = array();
    foreach ($parts as $part) {
      if ('.'  == $part) continue;
      if ('..' == $part) {
          array_pop($absolutes);
      } else {
          $absolutes[] = $part;
      }
    }
    $path=implode(DIRECTORY_SEPARATOR, $absolutes);
    return self::path( $path );
  }


  /**
   * This function is to replace PHP's extremely buggy realpath().
   * @param string The original path, can be relative etc.
   * @return string The resolved path, it might not exist.
   */
  public static function TruePath($path){
    // whether $path is unix or not
    $unipath=strlen($path)==0 || $path[0]!='/';
    // attempts to detect if path is relative in which case, add cwd
    if(strpos($path,':')===false && $unipath)
        $path=getcwd().DIRECTORY_SEPARATOR.$path;
    // resolve path parts (single dot, double dot and double delimiters)
    $path = str_replace(array('/', '\\'), DIRECTORY_SEPARATOR, $path);
    $parts = array_filter(explode(DIRECTORY_SEPARATOR, $path), 'strlen');
    $absolutes = array();
    foreach ($parts as $part) {
        if ('.'  == $part) continue;
        if ('..' == $part) {
            array_pop($absolutes);
        } else {
            $absolutes[] = $part;
        }
    }
    $path=implode(DIRECTORY_SEPARATOR, $absolutes);
    // resolve any symlinks
    // if(file_exists($path) && linkinfo($path)>0) $path=readlink($path);
    // put initial separator that could have been lost
    // $path=!$unipath ? '/'.$path : $path;
    $path='/'.$path;
    return self::path( $path );
  }


  


  public static function SecurePath(&$path){
    $path = self::TruePath(self::$BACK_END_FILE_MANAGER_ROOT."/".$path);    
    self::$BACK_END_FILE_MANAGER_ROOT = self::TruePath(self::$BACK_END_FILE_MANAGER_ROOT);
    if( self::startsWith($path,  self::$BACK_END_FILE_MANAGER_ROOT) ){
      return self::path( $path );
    }else{
      FileManager::ResponseError("Path is not in the sharing folder");
    }
  }

  public static function SecureRoot($path){
    return self::path( substr($path,strlen(self::$BACK_END_FILE_MANAGER_ROOT) ) );
  }

  /**
   * Returns $path parent folder
   */
  public static function PathFolder($path){
    $path = self::TruePath($path);
    $path = substr($path, 0, strrpos($path, DIRECTORY_SEPARATOR));
    if($path == "") $path = "/";
    return $path;
  }

  public static function PathFile($path){
    $path = self::TruePath($path);
    if( strrpos( $path , DIRECTORY_SEPARATOR ) ){
      $path = substr($path, strrpos($path, DIRECTORY_SEPARATOR)+1);
    }
    return $path;
  }

  public static function PathExtension($path){
    $file = self::PathFile($path);
    $pos = strrpos($file, ".");
    if($pos === false) return "";
    return substr($file, $pos+1);
    $ext = substr($file, strrpos($file, ".")+1);
    if($ext==$file) $ext="";
    return $ext;
  }

  public static function FileFromInode($inode, $basePath){
    $basePath = self::TruePath($basePath);
    // Get All Files in the folder
    $files = self::ListDirectory($basePath);
    foreach($files["files"] as $file){
      if( $file["inode"] == $inode) return self::BeatifulPath( $basePath . "/" . $file["name"] );
    }
    foreach($files["folders"] as $file){
      if( $file["inode"] == $inode) return self::BeatifulPath( $basePath . "/" . $file["name"] );
    }
    return false;
  }

  public static function Check($source){
    if(file_exists($source)){
        return true;
    }else if(is_dir($source)){
        return true;
    }
    return false;
  }

  public static function FolderSize($source){
    $size = 0;
    foreach(new RecursiveIteratorIterator(new RecursiveDirectoryIterator($source)) as $file){
        $size+=$file->getSize();
    }
    return $size;
  }

  public static function Properties($source){
    function formatBytes($size, $precision = 2){
      $base = log($size, 1000);
      $suffixes = array('', 'KB', 'MB', 'GB', 'TB');   
      return round(pow(1000, $base - floor($base)), $precision) .' '. $suffixes[floor($base)];
    }
    $io = popen ( '/usr/bin/du -sk ' . $source, 'r' );
    $size = fgets ( $io, 4096);
    $size = substr ( $size, 0, strpos ( $size, "\t" ) );
    pclose ( $io );
    return  formatBytes( $size * 1024 );
  }

  /** 
   * Response is
    ```{
    folders:[
      {name:"Folder",size:100,date},
    ],
    files :[
      {name:"File",size:100,date},
    ]
   }``
   */
  public static function ListDirectory($dirname){
    if (is_dir($dirname))
      $dir_handle = opendir($dirname);
    else
      return false;
    $dirList = [];
    $fileList = [];
    while($file = readdir($dir_handle)) {
        if ($file != "." && $file != "..") {
          if (!is_dir($dirname."/".$file)){
            $_file =[
              "name"=>$file,
              "size"=>@filesize($dirname."/".$file),
              "time"=>@filemtime($dirname."/".$file) * 1000,
              "inode"=>@fileinode($dirname."/".$file)
            ];
            if(self::$CREATE_THUMBNAIL && in_array(strtolower(pathinfo($file, PATHINFO_EXTENSION)),["jpg","jpeg","png","gif","webp"])){
              self::CreateThumbnail($dirname."/".$file);
              $__path__ = $dirname."/".$file;
              $folder = dirname($__path__);
              $file = basename($__path__);
              $size = min(self::$THUMBNAIL_WIDTH);
              $thumbnail = ".thumbnail/".$size."-".$file;
              if(self::endsWith($folder,".thumbnail")){
                $thumbnail = $file;
              }
              $_file["thumbnail"] = $thumbnail;
            }
            if(self::$CREATE_THUMBNAIL && strtolower(pathinfo($file, PATHINFO_EXTENSION))=="svg" && $_file["size"] < 5000 ){
              $_file["thumbnail"] = $file;
            }
            $fileList[] = $_file;
          }else{
            $_folder = [
              "name"=>$file,
              "size"=>"",
              "time"=>filemtime($dirname."/".$file) * 1000,
              "inode"=>fileinode($dirname."/".$file)
            ];
            if(self::$SHOW_FOLDER_SIZE){
              $_folder["size"] = self::FolderSize($dirname."/".$file);
            }
            $dirList[] = $_folder;
          }            
        }
    }
    return ["folders"=>$dirList,"files"=>$fileList];
  }

  //  $command="tree"
  public static function TreeFiles($path){
    $objects = new RecursiveIteratorIterator(new RecursiveDirectoryIterator($path), RecursiveIteratorIterator::SELF_FIRST);
    $files = [];
    foreach($objects as $name => $object){
      if ($object->isDir()) continue;
      $_name = substr("$name" , strlen($path)  ) ;
      if( !self::endsWith($_name,"/.") && !self::endsWith($_name,"/..") ){
        $files[] = substr($_name,1);
      }
    }
    return $files;
  }

  public static function TreeDirectory($source){
    $data = self::ListDirectory($source);
    $folders = $data["folders"];
    $files = $data["files"];
    $_folders = [];
    $_files = [];
    foreach($folders as $key => $value){
      $folder = $folders[$key];
      $folder["children"] = self::TreeDirectory($source."/".$folder["name"]);
      $_folders[] = $folder;
    }
    foreach($files as $key => $value){
      $file = $files[$key];
      $_files[] = $file;
    }
    return array("folders"=>$_folders,"files"=>$_files);
  }

  public static function ListAllFiles($source,$root=null){
    $source = self::TruePath($source);
    if($root==null){ $root = $source; }
    $data = self::ListDirectory($source);
    if($data){
      $folders = $data["folders"];
      $files = $data["files"];
      $response = [];
      foreach($files as $key => $file){
        $response[] = self::BeatifulPath( substr($source."/".$file["name"], strlen($root)) ); 
      }
      foreach($folders as $key => $folder){
        $response = array_merge($response,self::ListAllFiles($source."/".$folder["name"],$root));
      }
      return $response;
    }else{
      return [];
    }
  }

  public static function CreateFolder($path){
    if( !file_exists($path) ){
      mkdir( $path, 0777, true);
    }
    return $path;
  }



  public static function FindName($path){
    $path = self::TruePath($path);
    if( !file_exists($path) ){
      return $path;
    }else{
      $path_parts = pathinfo($path);
      $name = $path_parts["filename"];
      $ext = isset($path_parts["extension"]) ? $path_parts["extension"] : "";
      $i = 1;
      while( file_exists($path) ){
        if($ext==""){
          $path = $path_parts["dirname"]."/".$name."_".$i;
        }else{
          $path = $path_parts["dirname"]."/".$name."_".$i.".".$ext;
        }
        $i++;
      }
      return $path;
    }
  }

  /**
   * Move file to new path if file exist rename with timestamp
   */
  public static function Move($source,$target,$rename=false,$overwrite=false){    
    if($source==$target && $rename==false){
      return false;
    }
    
    if( file_exists($target) && $rename ){ // Rename
      $target = self::FindName($target);
    }else if( file_exists($target) && $overwrite ){ // Overwrite 
      //unlink($target);
    }else if( file_exists($target) ){ // Dont do anything
      return false;
    }
    if($source!=$target){
      if( $overwrite && is_dir($target) && is_dir($source)){
        // Merge
        $files = self::ListDirectory($source);
        foreach($files["files"] as $file){
          self::Move($source."/".$file["name"],$target."/".$file["name"],false,true);
        }
        foreach($files["folders"] as $folder){
          self::Move($source."/".$folder["name"],$target."/".$folder["name"],false,true);
        }
        rmdir($source);
      }else{
        // Remove Thumbnail
        $path = $source;
        $folder = self::PathFolder($path);
        $file = self::PathFile($path);
        $thumbnail = $folder."/.thumbnail/".$file;
        if(file_exists($thumbnail)){
          unlink($thumbnail);
        }
        rename($source,$target);
      }
      return true;
    }
    return false;
  }


  /**
   
  * If source and target is same name and rename is true, rename source to new name
   */
  public static function Copy($source,$target,$rename=false,$overwrite=false){
    //print_r([$source,$target,$rename,$overwrite])
    if($source==$target && $rename==true){
      $target = self::FindName($target);
      self::Copy($source,$target,false,$overwrite);
    }

    if(is_dir($source)) {
        if(!is_dir($target)){
          mkdir($target);
        }
        $dir_handle=opendir($source);
        while($file=readdir($dir_handle)){
          if($file!="." && $file!=".."){
            if(is_dir($source."/".$file)){
              if(!is_dir($target."/".$file)){
                mkdir($target."/".$file);
              }
              self::Copy($source."/".$file, $target."/".$file,false,$overwrite);
            } else {
              if( file_exists($target."/".$file)){ 
                if( $overwrite ){ // Overwrite 
                  unlink($target);
                  copy($source."/".$file, $target."/".$file);
                }// Else Dont do anything
              }else{ 
                copy($source."/".$file, $target."/".$file);
              }
            }
          }
        }
        closedir($dir_handle);
    } else {

      if( file_exists($target)){ 
        if( $overwrite ){ // Overwrite 
          unlink($target);
          copy($source, $target);
        }// Else Dont do anything
      }else{ 
        copy($source, $target);
      }

    }
  }



  
  // Remove File or Folder
  public static function Remove($path){
    if( file_exists($path) ){
      if( is_dir($path) ){
        $files = scandir($path);
        foreach ($files as $file) {
          if ($file != "." && $file != "..") {
            self::Remove($path."/".$file);
          }
        }
        rmdir($path);
      }else{
        // Remove Thumbnail
        $folder = self::PathFolder($path);
        $file = self::PathFile($path);
        $thumbnail = $folder."/.thumbnail/".$file;
        if(file_exists($thumbnail)){
          unlink($thumbnail);
        }
        // Remove File
        unlink($path);
      }
    }
  }



  public static function UploadedFiles($name){
    $files = [];
    
    if(!isset($_FILES[$name])){
      return $files;
    }

    $count = count($_FILES[$name]['name']);
    for($i=0; $i<$count; $i++){
      $file = array( 
        "name" => $_FILES[$name]["name"][$i],
        "type" => $_FILES[$name]["type"][$i],
        "tmp_name" => $_FILES[$name]["tmp_name"][$i],
        "error"=>$_FILES[$name]["error"][$i], 
        "size" => $_FILES[$name]["size"][$i],
      );
      $files[] = $file;
    }
    return $files;

  }

  /** 
   * Upload File to $folder if $file is in $folder rename 
   * @param string $FILE $_FILES[$file]
   * @param string $folder /$folder
   * @return string New File Path
  */
  public static function Upload($FILE,$folder,$name=null,$overwrite=false){
    self::CreateFolder( $folder );
    $fileName = $name==null ? $FILE["name"] : $name;
    if( $overwrite )
      $target = self::TruePath($folder."/".$fileName);
    else
      $target = self::FindName( $folder."/".$fileName); 
    
    if( move_uploaded_file($FILE["tmp_name"], $target) ){
      return $target;
    }else{
      self::ResponseJson(["status"=>"error","data"=>"Upload Error To:".$fileName]);
      return false;
    }
  }



  /**
  * Resize Image
  * @param $source - source file path
  * @param $target - target file path
  * @param $size - target image width
  *
  * #Example
  * $filePath = imageResize($_FILES['image']['tmp_name'], "upload/image1-1080.jpg", 1080);
  */
  public static function Resize($source, $target, $size = null,$_ext=null) {

    // Library Support Check (windows, linux) for imagecreatefromjpeg
    if (!function_exists("imagecreatefromjpeg")) {
        return false;
    }

    $targetExt = strtolower(pathinfo($target, PATHINFO_EXTENSION));
    $sourceExt = strtolower(pathinfo($source, PATHINFO_EXTENSION));

    // If the source is SVG, just copy it
    if ($sourceExt == "svg") {
        return self::Copy($source, $target);
    }

    // Set quality for different target image types
    $quality = 75; // Default quality for JPEG and WEBP
    if ($targetExt == "png") {
        $quality = 2; // Compression level for PNG (0-9, 0 being no compression)
    }

    // Validate image file
    if (!$imageSize = @getimagesize($source)) {
      return false;
    }

    $width = $imageSize[0];
    $height = $imageSize[1];
    $imageType = $imageSize[2];

    // Calculate target dimensions
    if ($size === null) {
        $size = $width;
    }
    $targetWidth = $size;
    $targetHeight = $size * $height / $width;
    if ($targetWidth > $width) {
        $targetWidth = $width;
        $targetHeight = $height;
    }

    // Create image resource from source
      
    switch ($imageType) {
      case IMAGETYPE_PNG:
          $imageResourceId = imagecreatefrompng($source); 
          break;
      case IMAGETYPE_GIF:
          $imageResourceId = imagecreatefromgif($source); 
          break;
      case IMAGETYPE_JPEG:
          $imageResourceId = imagecreatefromjpeg($source); 
          break;
      case IMAGETYPE_WEBP:
          $imageResourceId = imagecreatefromwebp($source); 
          break;
      default:
          return false;
          break;
    }


    // Create a true color image with the target dimensions
    $targetLayer = imagecreatetruecolor($targetWidth, $targetHeight);

    // Handle transparency for PNG and GIF
    if ($targetExt == "png" || $targetExt == "gif" || $targetExt == "webp") {
      imagealphablending($targetLayer, false);
      imagesavealpha($targetLayer, true);
      $transparency = imagecolorallocatealpha($targetLayer, 255, 255, 255, 127);
      imagefilledrectangle($targetLayer, 0, 0, $targetWidth, $targetHeight, $transparency);
    }

    // Resize the image
    imagecopyresampled($targetLayer, $imageResourceId, 0, 0, 0, 0, $targetWidth, $targetHeight, $width, $height);

    if($_ext) { $targetExt = $_ext; }
    // Save the resized image to the target path
    switch ($targetExt) {
        case "png":
            imagepng($targetLayer, $target, $quality);
            break;
        case "gif":
            imagegif($targetLayer, $target);
            break;
        case "jpeg":
        case "jpg":
            imagejpeg($targetLayer, $target, $quality);
            break;
        case "webp":
            if (!function_exists("imagewebp")) {
                return false;
            }
            imagewebp($targetLayer, $target, $quality);
            break;
        default:
            return false;
    }

    // Free up memory
    imagedestroy($imageResourceId);
    imagedestroy($targetLayer);

    return $target;
  }


  public static function CreateThumbnail($source){
    $folder = dirname($source);
    $file = basename($source);

    if(self::endsWith($folder,"/.thumbnail")){
      return $file;
    }

    $thumbnailFolder = self::CreateFolder( $folder."/.thumbnail" );
    foreach(self::$THUMBNAIL_WIDTH as $width){
      $thumbnail = $thumbnailFolder."/".$width."-".$file;
      if( !file_exists($thumbnail) ){
        self::Resize($source,$thumbnail,$width,self::$THUMBNAIL_FORMAT);
      }
    }
    return "/.thumbnail/".$file;
  }



  public static function Zip($sources,$target){
    $zip = new ZipArchive();
    if ($zip->open($target, ZIPARCHIVE::CREATE) === TRUE) {
      foreach ($sources as $source) {
        if(is_dir($source)){
          $files = new RecursiveIteratorIterator(
            new RecursiveDirectoryIterator($source),
            RecursiveIteratorIterator::LEAVES_ONLY
          );
          
          foreach ($files as $name => $file){
            // Skip directories (they would be added automatically)
            if (!$file->isDir())
            {
                // Get real and relative path for current file
                $filePath = $file->getRealPath();
                $relativePath = substr($filePath, strlen($source) + 1);
                // Add current file to archive
                $zip->addFile($filePath, basename($source) . "/" . $relativePath);
            }
          }
        }else{
          $zip->addFile($source,basename($source));
        }
      }
      $zip->close();
      return true;
    }
    return false;
  }

  public static function Unzip($source,$target){
    $zip = new ZipArchive;
    $res = $zip->open($source);
    if ($res === TRUE) {
      $zip->extractTo($target);
      $zip->close();
      return true;
    }
    return false;
  }


  public static function get($key, $default = null) {
    return isset($_GET[$key]) ? $_GET[$key] : $default;
  }

  public static function post($key, $default = null) {
    return isset($_POST[$key]) ? $_POST[$key] : $default;
  }

  public static function all($key, $default = null) {
    return isset($_REQUEST[$key]) ? $_REQUEST[$key] : $default;
  }
  
  public static function startsWith($text,$search){
    return substr($text, 0, strlen($search)) === $search;
  }
  public static function endsWith($text,$search){
    return substr($text, -strlen($search)) === $search;
  }


  public static function DefineOptions($options=[]){
    if( isset($options["BACK_END_FILE_MANAGER_ROOT"]) ){ self::$BACK_END_FILE_MANAGER_ROOT = $options["BACK_END_FILE_MANAGER_ROOT"]; }
    if( isset($options["MASTER_URL"]) ){ self::$MASTER_URL = $options["MASTER_URL"]; }
    if( isset($options["CREATE_THUMBNAIL"]) ){ self::$CREATE_THUMBNAIL = $options["CREATE_THUMBNAIL"]; }
    if( isset($options["THUMBNAIL_WIDTH"]) ){ self::$THUMBNAIL_WIDTH = $options["THUMBNAIL_WIDTH"]; }
    if( isset($options["THUMBNAIL_FORMAT"]) ){ self::$THUMBNAIL_FORMAT = $options["THUMBNAIL_FORMAT"]; }
    if( isset($options["SHOW_FOLDER_SIZE"]) ){ self::$SHOW_FOLDER_SIZE = $options["SHOW_FOLDER_SIZE"]; }
    if( isset($options["SECURITY_ENABLED"]) ){ self::$SECURITY_ENABLED = $options["SECURITY_ENABLED"]; }
    if( isset($options["SECURITY_USERS"]) ){ self::$SECURITY_USERS = $options["SECURITY_USERS"]; }
    if( isset($options["LOGIN_HASH"]) ){ self::$LOGIN_HASH = $options["LOGIN_HASH"]; }
    if( isset($options["LOGIN_NAME"]) ){ self::$LOGIN_NAME = $options["LOGIN_NAME"]; }
    if( isset($options["LOGIN_TIME"]) ){ self::$LOGIN_TIME = $options["LOGIN_TIME"]; }
    if( isset($options["READ_ONLY"]) ){ self::$READ_ONLY = $options["READ_ONLY"]; }
    if( isset($options["EXTENSIONS"]) ){ self::$EXTENSIONS = $options["EXTENSIONS"]; }

    self::$BACK_END_FILE_MANAGER_ROOT = self::TruePath( self::$BACK_END_FILE_MANAGER_ROOT );    
    if(!is_array(self::$THUMBNAIL_WIDTH)){
      self::$THUMBNAIL_WIDTH=[self::$THUMBNAIL_WIDTH];
    }
  }

  public static function Encode($text){
    return base64_encode($text);
  }

  public static function Decode($code){
    return base64_decode($code);
  }

  public static function SecureExtension($path){
    if(self::$EXTENSIONS=="*") return true;
    $extensions = explode(",",self::$EXTENSIONS);
    return in_array(self::PathExtension($path),$extensions);
  }


  public static function Run_FileManager( $options = [] ){
    self::DefineOptions($options);
    
    if( FileManager::all("action")=="command" ){
      $command = FileManager::all("command");
      $parameters = explode(",",FileManager::all("parameters")) ;
      $parameters = array_map(function($item){ return self::Decode($item); },$parameters);
      // dir [parameter]
      if(  FileManager::startsWith($command,"dir") ){
        $path = FileManager::SecurePath( $parameters[0] );
        $root = FileManager::SecureRoot( $path ).DIRECTORY_SEPARATOR ;
        $list = FileManager::ListDirectory( $path  );
        $inode = file_exists($path) ? fileinode( $path ) : "-1";
        if($list){
          FileManager::ResponseSuccess(array( "path"=> self::linuxPath($root) ,"inode"=>$inode, "list" => $list ) );
        }else{
          FileManager::ResponseJson([
            "status"=>"error",
            "data"=>"Invalid Path",
            "path"=>$parameters[0],
          ]);
        }
      }


      

      // tree [parameter]
      if(  FileManager::startsWith($command,"tree") ){
        $path = FileManager::SecurePath( $parameters[0] );
        $root = FileManager::SecureRoot( $path ).DIRECTORY_SEPARATOR ;
        $list = FileManager::TreeFiles( $path  );
        $inode = file_exists($path) ? fileinode( $path ) : "-1";
        if($list){
          FileManager::ResponseSuccess(array( "path"=> self::linuxPath($root) ,"inode"=>$inode, "list" => $list ) );
        }else{
          FileManager::ResponseJson([
            "status"=>"error",
            "data"=>"Invalid Path",
            "path"=>$parameters[0],
          ]);
        }
      }
    
      // New Folder 
      if(  FileManager::startsWith($command,"mkdir") && self::$READ_ONLY==false){
        FileManager::ResponseSuccess(FileManager::CreateFolder( FileManager::SecurePath( $parameters[0] ) ));
      }
  
      // New File
      if(  FileManager::startsWith($command,"touch") && self::$READ_ONLY==false && self::SecureExtension($parameters[0])){
        $path = FileManager::SecurePath( $parameters[0] );
        if( file_exists($path) ){
          FileManager::ResponseError("File already exists");
        }else{
          FileManager::ResponseSuccess( file_put_contents($path,"") );
        }
      }

      
      // New File
      if(  FileManager::startsWith($command,"convert") ){
        $path = FileManager::SecurePath( $parameters[0] );
        
        $type = $parameters[1];
        //$type = "webp";
        $target = $path . "." . $type;

        $size = $parameters[2];
        $size = null;
        
        self::Resize($path, $target, $size, $type); 
        FileManager::ResponseSuccess( $target );
      }
    
    
      // Remove File or Folder
      if( FileManager::startsWith($command,"rm") && self::$READ_ONLY==false){
        // If $BACK_END_FILE_MANAGER_ROOT folder have not .trash folder recreate .trash folder
        FileManager::CreateFolder( FileManager::$BACK_END_FILE_MANAGER_ROOT . DIRECTORY_SEPARATOR  . ".trash" );
        $force = FileManager::post("force",0);
        $removedCount = 0;
        foreach( $parameters as  $parameter ){
          
          $path = FileManager::SecurePath( $parameter );
          $rootPath = FileManager::SecureRoot( $parameter );
    
          if($rootPath==""){
            FileManager::ResponseError("Can't remove root folder");
          }
          // Remove thumbnails
          $folder = dirname($path);
          $file = basename($path);
          foreach(self::$THUMBNAIL_WIDTH as $width){
            FileManager::Remove( $folder . DIRECTORY_SEPARATOR . ".thumbnail" . DIRECTORY_SEPARATOR . $width ."-" .$file ) ;  
          }

          // If in .trash remove
          $trashPath = DIRECTORY_SEPARATOR . ".trash" . DIRECTORY_SEPARATOR ;;
          if( FileManager::startsWith( $rootPath , $trashPath ) && $rootPath !=  $trashPath   && $rootPath !=  DIRECTORY_SEPARATOR. ".trash"  ){
            FileManager::Remove( $path ) ;
            $removedCount++;
          }
          
          // If force
          if( $force == "1" ){
            FileManager::Remove( $path );
            $removedCount++;
          }
          // Else move to .trash
          else{
            FileManager::Move( $path , FileManager::$BACK_END_FILE_MANAGER_ROOT . $trashPath . basename($path) , true );
            $removedCount++;
          }
        }
        FileManager::ResponseSuccess(array( "removedCount" => $removedCount ));    
    
      }
    
      // Rename or Move
      if( FileManager::startsWith($command,"mv") && self::$READ_ONLY==false && self::SecureExtension($parameters[0]) && self::SecureExtension($parameters[1])){
        $force = substr($command,2,2)=="-f";
    
        $source = FileManager::SecurePath( $parameters[0] );
        $target = FileManager::SecurePath( $parameters[1] );
    
        $response =  FileManager::Move( $source , $target , false , $force );
        if($response){
          FileManager::ResponseSuccess("File moved");
        }else{
          FileManager::ResponseError("Can't move file");
        }
      }
    
      if( FileManager::startsWith($command,"zip") ){
        $sources = []; 
        $target="";
        foreach( $parameters as $key => $parameter ){
          if( $key==count($parameters)-1 ){
            $target = FileManager::SecurePath( $parameter );
          }else{
            $sources[] = FileManager::SecurePath( $parameter );
          }
        }
        $target = FileManager::FindName( $target );
        $response =  FileManager::Zip( $sources , $target );
        if($response){
          FileManager::ResponseSuccess( FileManager::SecureRoot( $target ) );
        }else{
          FileManager::ResponseError("Can't zip file");
        }
      }
    
      if( FileManager::startsWith($command,"unzip") && self::$READ_ONLY==false){
        $source = FileManager::SecurePath( $parameters[0] );
        $target = FileManager::SecurePath( $parameters[1] );
        $target = FileManager::FindName( $target ); 
        $response =  FileManager::Unzip( $source , $target );
        if($response){
          FileManager::ResponseSuccess("File unzipped");
        }else{
          FileManager::ResponseError("Can't unzip file");
        }
      }
    
      // Copy
      if( FileManager::startsWith($command,"cp") && self::$READ_ONLY==false){
        $force = FileManager::endsWith($command,"-f");
        $source = FileManager::SecurePath( $parameters[0] );
        $target = FileManager::SecurePath( $parameters[1] );
        $response =  FileManager::Copy( $source , $target , ! $force , $force );
        FileManager::ResponseSuccess("File(s) Copied");
      }
    
      // Upload 
      if( FileManager::startsWith($command,"upload") && self::$READ_ONLY==false && self::SecureExtension($parameters[0])){
        $target = FileManager::SecurePath( $parameters[0] );
        $files = FileManager::UploadedFiles("upload");
        $response = [];
        $replace = FileManager::post("replace")=="true";
        foreach( $files as $file ){
          $response[] =  substr( FileManager::Upload( $file , $target , null, $replace ) , strlen($target) + 1 );
        }
        FileManager::ResponseSuccess($response);
      }
      
      // image
      if( FileManager::startsWith($command,"image") ){
        $path = FileManager::SecurePath( $parameters[0] );
        $folder = dirname($path);
        $file = basename($path);
        $size = min(self::$THUMBNAIL_WIDTH);
        $thumbnail = $folder."/".$size."-".$file;
        if(self::endsWith($folder,".thumbnail")){
          $thumbnail = $path;
        }
        if(self::endsWith($file,".svg")){
          $thumbnail = $path;
        }
        self::ResponseImage($thumbnail);
      }

      // Edit
      if( FileManager::startsWith($command,"cat") ){
        $path = FileManager::SecurePath( $parameters[0] );
        $content = file_get_contents( $path );
        FileManager::ResponseSuccess(array( "path"=> $path, "content" => $content ));
      }
      
      // Download
      if( FileManager::startsWith($command,"download") ){
        $path = FileManager::SecurePath( $parameters[0] );
        self::ResponseFile($path);
      }
    
      // Save
      if( FileManager::startsWith($command,"save") && self::$READ_ONLY==false && self::SecureExtension($parameters[0]) ){
        $path = FileManager::SecurePath( $parameters[0] );
        $content = FileManager::post("content","");
        // Base64 decode
        $content = FileManager::Decode($content);
        $content = str_replace("\r\n", "\n", $content);
        $status = file_put_contents( $path , $content );
        if($status || $content==""){
          FileManager::ResponseSuccess("File Saved");
        }else{
          FileManager::ResponseError("Can't save file");
        }
      }
      
      // properties
      if( FileManager::startsWith($command,"properties") ){
        $path = FileManager::SecurePath( $parameters[0] );
        FileManager::ResponseSuccess(self::Properties($path));
        
        // Base64 decode
        $content = FileManager::Decode($parameters);
        
      }
      
      if(self::$READ_ONLY){
        self::ResponseSuccess("no action");
      }
      
    
    }
    
  }


  public static function Export_Template($path){
    $_list = self::ListAllFiles($path);
    $list = [];
    // $file -> /$file
    foreach( $_list as $key => $_file ){
      $file = "/".$_file;
      if( self::endsWith($file,".html") && strpos($file,".thumbnail/")===false  ){
        $list[] = substr($file,0,-5);
      }  
    }
    
    $tree = [];
    for($i=0;$i<count($list);$i++){
      $folder_name = self::PathFolder($list[$i]);
      if( $folder_name=="" ) $folder_name = "/";
      $file_name = $folder_name;
      if( !array_key_exists($folder_name,$tree) ){
        $tree[$folder_name] = [];
      }
      $file_name = self::PathFile($list[$i]);
      array_push($tree[$folder_name],$file_name);
    }
    return $tree;
  }



  


  public static function ResponseRedirect($page){
    $pref = $_SERVER["REQUEST_SCHEME"];
    $host = $_SERVER["HTTP_HOST"];        
    $url = $host."/".self::$MASTER_URL."/".$page;
    $url = str_replace("//","/",$url);
    $url = $pref."://".$url;
    header("Location: ".$url);
    exit();
  }

  public static function CssToJs($code){
    return "//-- Css --//
    (function(){
      let css = `$code`
      let style = document.createElement('style')
      style.innerHTML = css
      document.head.appendChild(style)
    })();";
  }


  public static function Run_Security($options=[]){
    self::DefineOptions($options);
    /*
      page=login
      page=logout
      page=index
      page=file-manager
      page=web-editable
      page=documentation
    */
    
    
    if(self::get("page")=="login"){

      if( self::isLogged() || self::$SECURITY_ENABLED==false ){
        exit( header("Location: /?page=index") );
      }
      
      $username = FileManager::post("username");
      $password = FileManager::post("password");
      if( isset(self::$SECURITY_USERS[$username]) && self::$SECURITY_USERS[$username]===$password ){
        self::login( $username );
        exit( header("Location: /?page=index") );
      }else{
        $message = "No Access";
        eval('?>' . self::file_get_contents(__DIR__ . "/pages/login.html") );
      }
      exit();
    }


    // Static css , js files and, help
    if(self::get("page")){
      $file = self::get("page");
      $extension = pathinfo($file, PATHINFO_EXTENSION);
      if( $file=="help" ){
        self::ResponseType("help.html");
        echo self::file_get_contents(__DIR__."/pages/help.html");
        exit;
      }
      if( $extension=="css" || $extension=="js" ){
        self::ResponseType($file);
        eval('?>' . self::file_get_contents(__DIR__ . "/pages/".$file) );
        exit();
      }
    }

    if( !self::isLogged() && self::$SECURITY_ENABLED==true ){
      $base = $_SERVER["SCRIPT_URI"];
      $base = strpos($base,"/folder/")!==false ? substr($base, 0, strpos($base,"/folder/")) . "/" : "";
      exit( header("Location: ".$base."?page=login") );
    }
    $username = self::id();

    
    // logout
    if( self::get("page")=="logout" ){
      self::logout();
      exit( header("Location: ?page=login") );
    }

    if(self::get("page")=="file-manager-library"){
      
      self::ResponseType(".js");
      foreach(["FileManager.js","Icons.js","ProgressBar.js"] as $file){
        echo self::file_get_contents("pages/".$file) . ";\n;";
      }
      echo "(function(){ let css=`";
      echo self::file_get_contents("pages/ModalView.css") . "\n\n";
      //echo self::file_get_contents("pages/ProgressBar.css") . "\n\n";
      echo "`;let style=document.createElement('style');style.innerHTML=css;document.head.appendChild(style);";
      echo "})()";
      exit();
    }


    if(self::get("page")){
      $file = self::get("page");
      $extension = pathinfo($file, PATHINFO_EXTENSION);
      if($extension=="") $file .= ".html";
      self::ResponseType($file);
      eval('?>' . self::file_get_contents(__DIR__ . "/pages/".$file) );
      exit();
    }
  }

  public static $LOGIN_HASH = 'RAND0M-T€XT-F0R-5€5510N'; // Random Text
  public static $LOGIN_NAME = 'master-session';
  public static $LOGIN_TIME = 3600 * 24 * 360; // 1 Year
  
  public static function Run_All($options=[]){
    self::DefineOptions($options);
    self::Run_Security();
    self::Run_FileManager();
    exit( header("Location: ?page=index") );
  }
  
  public static function login($id){
    if(strpos($id,"___")!==false) {
      echo "ID should not contain '___'";
      exit();
    }
    $hash = [
      $id,
      md5(self::$LOGIN_HASH.md5($id.self::$LOGIN_HASH)),
      time() + self::$LOGIN_TIME,
      md5(self::$LOGIN_HASH.md5( time() + self::$LOGIN_TIME . self::$LOGIN_HASH ))
    ];
    setcookie( self::$LOGIN_NAME   ,json_encode($hash), time() + self::$LOGIN_TIME , "/" );
  }

  public static function logout(){
    unset($_COOKIE[self::$LOGIN_NAME]); 
    setcookie(self::$LOGIN_NAME, '[]', -1 , "/"); 
  }

  public static function isLogged(){
    if( self::id()==false ) return false;
    if( isset(self::$SECURITY_USERS[self::id()]) ) return true;
    return false;
  }

  public static function id(){
    if(! isset($_COOKIE[self::$LOGIN_NAME]) ) return false;
    
    $hash = [];
    $hash = @json_decode($_COOKIE[self::$LOGIN_NAME],true);
    if(!$hash) return false;
    if( count($hash) < 4 ) return false;

    $id          = $hash[0];
    $id_verify   = $hash[1];
    $time        = $hash[2];
    $time_verify = $hash[3];

    if( $id_verify != md5(self::$LOGIN_HASH.md5($id.self::$LOGIN_HASH)) ) return false;
    if( $time_verify != md5(self::$LOGIN_HASH.md5($time.self::$LOGIN_HASH)) ) return false;
    if(time() > $time) return false;

    return $id;
  }

}



FileManager::$LOGIN_HASH = 'RAND0M-T€XT-F0R-5€5510N'.$_SERVER['DOCUMENT_ROOT']; // Random Text
FileManager::$LOGIN_NAME = strtolower(trim(preg_replace('/[^A-Za-z0-9-]+/', '-', $_SERVER['SERVER_NAME'].'-master-session' )));
FileManager::$LOGIN_TIME = 3600 * 24 * 360; // 1 Year
