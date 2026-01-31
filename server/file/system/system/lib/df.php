<?php


/*
Dinamic DF object
*/
class DF{


    /**
     * Create new DF object
     * @file -> absolute path or document inside file
     */
    function __construct( $file=".htdatabase.json" ) {
      $this->file = $file;
    }

    private $file=".htdatabase.json";

    public $db=null;

    public $__ID_NAME__ ="id";

    private function dbPath(){
      return $this->file;
      if( !startsWith($this->file,"/") ){
        return $_SERVER["DOCUMENT_ROOT"] ."/". $this->file;
      }else{
        return $this->file;
      }
    }

    private function db(){
        $full_path = $this->dbPath();
        if($this->db==null){
            if(!is_file($full_path)){
                file_put_contents($full_path,"{}");
            }
            $this->db = JSON::Parse( file_get_contents($full_path) );
        }
    }

    private function &store($name){
        $this->db();
        if( array_key_exists($name,$this->db) ){
            return $this->db[$name];
        }else{
            $this->db[$name] = array("lastId"=>0,"datas"=>[]);
            return $this->db[$name];
        }
    }

  
    public function save(){
        $time = &$this->refSet("_db/lastUpdate" );
        $time = date('d M Y H:i:s');
        file_put_contents( $this->dbPath() ,JSON::ReadableString($this->db));
    }



    /**
      * Using: $value = &refGet("path");
      */
    public function &refGet($path,$default=null){
        $this->db();
        $data = &$this->db;
        $founded = false;
        foreach( explode("/",$path) as $p ){
            // p start with
            if( !is_array($data) ){
                $data = array();
            }
            if( array_key_exists($p,$data) ){
                $data = &$data[$p];
                $founded=true;
            }else{
                $data[$p] = array();
                $data = &$data[$p];
                $founded=false;
            }
        }

        if($founded==false){
            $data = $default;
        }
        return $data;
    }

    /**
      * Using: $value = &refSet("path");
      */
    public function &refSet($path,$value=null){
        $this->db();
        $data = &$this->db;
        $length = count(explode("/",$path));
        foreach( explode("/",$path) as $index=>$p ){
            // p start with
            if( !is_array($data) ){
                $data = array();
            }
            if( array_key_exists($p,$data) ){
              if($index==$length-1){
                $data[$p] = $value;
                return $data[$p];
              }
              $data = &$data[$p];
            }else{
              $data[$p] = array();
              $data = &$data[$p];
            }
        }

        if($value!=null){
            $data = $value;
        }
        return $data;
    }

    public function refRemove($path){
        $this->db();
        $data = &$this->db;

        $_path = explode("/",$path);
        $_last = array_pop($_path);
        $data = &$this->db;
        foreach( $_path as $p ){
            // p start with
            if( !is_array($data) ){
                $data = array();
            }
            if( array_key_exists($p,$data) ){
                $data = &$data[$p];
            }else{
                $data[$p] = array();
                $data = &$data[$p];
            }
        }
        if( array_key_exists($_last , $data) ){
            unset( $data[$_last] );
            return true;
        }
        return false;
    }


    public function &add($name,$data){
        $store = &$this->store($name);
        $newId = ++$store["lastId"];
        $data = array_merge(array( $this->__ID_NAME__ =>$newId),$data);
        $store["datas"][$newId] = $data;
        $this->save();
        return $this->get($name,$newId);
    }

    public function &get($name,$where=null){
        $_FALSE_ = false;
        $store = &$this->store($name);
        if($where==null){
            if( count($store["datas"]) > 0 ){
                return $store["datas"][0];
            }
            return $_FALSE_;
        }else if( (int) $where == $where ){
            if(array_key_exists($where,$store["datas"])){
                return $store["datas"][$where];
            }
            return $_FALSE_;
        }else if(is_array($where)){
            $rows = [];
            foreach ($store["datas"] as $row) {
                $condution=true;
                foreach ($where as $key => $value) {
                    if( !is_array($row) ){
                        $condution=false;
                        break;
                    }
                    if( !array_key_exists($key,$row)){
                        $condution=false;
                        break;
                    }
                    if($row[$key]!=$value){
                        $condution=false;
                        break;
                    }
                }
                if($condution)
                    return $row;
            }
            return $_FALSE_;
        }
        return $_FALSE_;
    }



    public function set($name,$data){
      $_data = &$this->get($name,$data["id"]);
      if($_data===false){
          return false;
      }
      foreach ($data as $key => $value) {
          $_data[$key] = $value;
      }
      $this->save();
}


    public function &all($name,$where=null){
        $store = &$this->store($name);
        if($where==null){
            return $store["datas"];
        }else if(is_array($where)){
            $rows = [];
            foreach ($store["datas"] as $row_id => $row) {
                $condution=true;
                foreach ($where as $key => $value) {
                    if($row[$key]!=$value){
                        $condution=false;
                        break;
                    }
                }
                if($condution)
                    $rows[$row_id] = &$store["datas"][$row_id];
            }
            return $rows;
        }
        return false;
    }

    /**

    @returns {int} How many item removed.
    */
    public function remove($name,$where){
        $store = &$this->store($name);
        if($where===true){
            $count = count($store["datas"]);
            $store["datas"] = [];
            $this->save();
            return $count;
        }else if( (int) $where == $where ){
            if(array_key_exists($where,$store["datas"])){
                unset($store["datas"][$where]);
                $this->save();
                return 1;
            }
            return 0;
        }else if(is_array($where)){
            $count = 0;
            foreach ($store["datas"] as $row_key => $row) {
                $condution=true;
                foreach ($where as $key => $value) {
                    if($row[$key]!=$value){
                        $condution=false;
                        break;
                    }
                }
                if($condution){
                    $count++;
                    unset($store["datas"][$row_key]);
                }
            }
            $this->save();
            return $count;
        }
        return 0;
    }


    /**
      * Change order selected $store
      * @order => array("-404","1","2")
      */
    public function order($order,&$store=null){
        $this->db();
        if($store==null) $store = &$this->db;

        $keys = array_keys($store);
        $new_store = [];
        foreach($order as $key){
            if(array_key_exists($key,$store)){
                $new_store[$key] = $store[$key];
            }
        }
        foreach($store as $key=>$value){
            if(!array_key_exists($key,$new_store)){
                $new_store[$key] = $store[$key];
            }
        }
        $store = $new_store;
        return $store;
    }


}




function startsWith ($string, $startString){
  $len = strlen($startString);
  return (substr($string, 0, $len) === $startString);
}
function endsWith($string, $endString){
  $len = strlen($endString);
  if ($len == 0) {
      return true;
  }
  return (substr($string, -$len) === $endString);
}





// ###################   JSON    ####################################




class Json{
    public static function Parse($string){
        return json_decode($string,true);
    }
    public static function String($array){
        return json_encode($array);
    }
    public static function ReadableString($array){
        return jsonToReadable( json_encode($array) );
    }
};

function jsonToReadable($json){
    $tc = 0;        //tab count
    $r = '';        //result
    $q = false;     //quotes
    $t = "\t";      //tab
    $nl = "\n";     //new line

    for($i=0;$i<strlen($json);$i++){
        $c = $json[$i];
        if($c=='"' && $json[$i-1]!='\\') $q = !$q;
        if($q){
            $r .= $c;
            continue;
        }
        switch($c){
            case '{':
            case '[':
                $r .= $c . $nl . str_repeat($t, ++$tc);
                break;
            case '}':
            case ']':
                $r .= $nl . str_repeat($t, --$tc) . $c;
                break;
            case ',':
                $r .= $c;
                if($json[$i+1]!='{' && $json[$i+1]!='[') $r .= $nl . str_repeat($t, $tc);
                break;
            case ':':
                $r .= $c . ' ';
                break;
            default:
                $r .= $c;
        }
    }
    return $r;
}

