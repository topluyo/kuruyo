<?php

/**
 

 */
class DB{

  public $host;
  public $user;
  public $pass;
  public $database;
  public $db;
  public $error = false;
  public $code = "";

  public function __construct(string $host = null, string $database = null, string $user = null, string $pass = null){
    $this->host = $host;
    $this->user = $user;
    $this->pass = $pass;
    $this->database = $database;
    $this->db = new mysqli($this->host, $this->user, $this->pass, $this->database);
    $this->db->set_charset("utf8mb4");
  }

  private function where($where){
    $_where = "";
    $_values = [];
    // If where type is number
    if(is_numeric($where)){
      $_where = "id = ?";
      $_values = [$where];
    }
    // If where type is array
    if(is_array($where)){
      $_where = "";
      $_isFirst = true;
      foreach($where as $key => $value){
        $_values[] = $value;
        if($_isFirst){
          $_where .= "`$key` = ? ";
        }else{
          $_where .= "and `$key` = ? ";
        }
        $_isFirst = false;
      }
    }
    return [$_where, $_values];
  }

  public function remove(string $table, $where=null){
    $query = "DELETE FROM `$table`";
    $_where_values = $this->where($where);
    $_where = $_where_values[0];
    $_values = $_where_values[1];

    if($_where != ""){
      $query .= " WHERE $_where";
    }
    return  $this->sql($query,$_values);
  }

  public function get(string $table, $where=null){
    $query = "SELECT * FROM `$table`";
    $_where_values = $this->where($where);
    $_where = $_where_values[0];
    $_values = $_where_values[1];

    
    if($_where != ""){
      $query .= " WHERE $_where";
    }
    $query .= " LIMIT 1";

    $result = $this->sql($query,$_values);
    if(count($result)==1) return $result[0];
    return false;
  }

  public function all(string $table, $where=null){
    $query = "SELECT * FROM `$table`";
    $_where_values = $this->where($where);
    $_where = $_where_values[0];
    $_values = $_where_values[1];

    if($_where != ""){
      $query .= " WHERE $_where";
    }
    
    
    $stmt = $this->db->prepare($query);
    // Check _where is empty
    if($_where != ""){
      $stmt->bind_param("".str_repeat("s", count($_values)), ...$_values);
    }
    $stmt->execute();
    $result = $stmt->get_result();
    $rows = [];
    while($row = $result->fetch_assoc()){
      $rows[] = $row;
    }
    $stmt->close();
    return $rows;
  }

  public function sql($sql,$parameters=[]){
    $sql = trim($sql);
    if($sql=="") return 0;

    
    $action = explode(" ",trim(strtolower(str_replace(["(",")"], "", str_replace("  "," ",$sql) ))) )[0];

    if($action=="create" || $action=="alter"){
      if ($this->db->query($sql) === TRUE) {
        return 1;
      } else {
        return $this->error = $this->db->error;
      }
    }

    $stmt = $this->db->prepare($sql);
    
    // If parameters is array
    if (is_array($parameters) && count($parameters) > 0) {
      if(domain()=="alfa.topluyo.com"){
        //write($parameters);
        //write($sql);
      }
      $stmt->bind_param("".str_repeat("s", count($parameters)), ...$parameters);

      // Generate the final SQL with values
      $boundSql = $sql;
      foreach ($parameters as $param) {
        $boundSql = preg_replace('/\?/', "'" . $this->db->real_escape_string($param) . "'", $boundSql, 1);
      }

      $this->code = $boundSql; // Store the final SQL query
    } else {
      $this->code = $sql; // If no parameters, just store the raw SQL
    }
    
    mysqli_report(0);

    if ($stmt->execute()) { 
      $this->error = false;
    } else {
      $this->error = $stmt->error;
      //! DEBUG
      file_put_contents( __DIR__."/log.txt", domain() . "::" . $this->error . "::" . $this->code . "\n"  ,FILE_APPEND );
      if(domain()=="alfa.topluyo.com"){
        die( domain() . "::" . $this->error . "::" . $this->code );
      }
      
    }


    // If Select Statement
    if( $action == "select" ){
     
      $result = [];
      $stmt->execute();

      // Get the result metadata
      $meta = $stmt->result_metadata();
      
      // Prepare an array to hold column names and bind variables
      $fields = [];
      $boundParams = [];
      $params = [];

      // Fetch column metadata
      while ($field = $meta->fetch_field()) {
          $fields[] = $field->name;
          $boundParams[$field->name] = null; // Initialize null values for binding
          $params[] = &$boundParams[$field->name]; // Create array of references
      }

      // Bind result variables
      call_user_func_array([$stmt, 'bind_result'], $params);

      // Fetch results
      while ($stmt->fetch()) {
          $row = [];
          foreach ($fields as $field) {
              $row[$field] = $boundParams[$field];
          }
          $result[] = $row;
      }

      // Close the statement
      $stmt->close();

      return $result;
      
    }

    // Update or Delete Statement
    if( $action=="update" ||  $action=="delete" ||  $action=="replace"){
      $row_counts = $stmt->affected_rows;
      $stmt->close();
      return $row_counts;
    }

    if($action=="insert"){
      $id = $stmt->insert_id;
      $stmt->close();
      return $id;
    }

    $stmt->close();
  }

  public function set($table,$parameters){
    $keys = [];
    $values = [];
    $equals = [];
    $marks  = [];
    foreach($parameters as $key => $value){
      $keys[]   = $key;
      $values[] = $value;
      $marks[]  = "?";
      $equals[] = " `$key`='". addslashes($value) ."' ";
    }
    $equalsSql = "".implode(",", $equals)."";
    $query = "INSERT INTO `$table` SET " . $equalsSql . " ON DUPLICATE KEY UPDATE " . $equalsSql;
    //$query = "REPLACE INTO $table (" . implode(",", $keys) . ") VALUES (". implode(",", $marks) .")";
    return $this->sql($query);
    // return $this->sql($query,$values);
  }



  public function def($table, $parameters) {
    $keys = [];
    $values = [];
    $equals = [];
    $marks  = [];
    $idExists = isset($parameters['id']); // Check if 'id' exists

    foreach ($parameters as $key => $value) {
        $keys[] = $key;
        $values[] = $value;
        $marks[] = "?";

        if (is_numeric($value)) {
            $equals[] = " `$key`=" . addslashes($value) . " ";
        } else {
            $equals[] = " `$key`='" . addslashes($value) . "' ";
        }
    }

    if ($idExists) {
        // Use UPDATE statement
        $id = addslashes($parameters['id']);
        unset($parameters['id']); // Remove 'id' from update fields
        $equalsSql = implode(",", $equals);
        $query = "UPDATE `$table` SET $equalsSql WHERE `id`='$id'";
    } else {
        // Use INSERT statement
        $columns = implode("`, `", array_keys($parameters));
        $values  = implode("', '", array_map('addslashes', array_values($parameters)));
        $query = "INSERT INTO `$table` (`$columns`) VALUES ('$values')";
    }

    return $this->sql($query);
  }




  public function definex($table,$parameters){
    $keys = [];
    $values = [];
    $equals = [];
    $marks  = [];
    foreach($parameters as $key => $value){
      $keys[]   = $key;
      $values[] = $value;
      $marks[]  = "?";
      if(is_numeric($value)){
        $equals[] = " `$key`=". addslashes($value) ." ";
      }else{
        $equals[] = " `$key`='". addslashes($value) ."' ";
      }
    }
    $equalsSql = "".implode(",", $equals)."";
    $query = "INSERT INTO `$table` SET " . $equalsSql . " ON DUPLICATE KEY UPDATE " . $equalsSql;
    //$query = "REPLACE INTO $table (" . implode(",", $keys) . ") VALUES (". implode(",", $marks) .")";
    return $this->sql($query);
    // return $this->sql($query,$values);
  }

  public function add($table, $parameters) {
    $keys = array_keys($parameters); // Get column names
    $values = array_values($parameters); // Get values
    $placeholders = array_fill(0, count($keys), '?'); // Create placeholders for prepared statements

    $columnsSql = "`" . implode("`, `", $keys) . "`"; // Format column names
    $placeholdersSql = implode(", ", $placeholders); // Format placeholders

    $query = "INSERT INTO `$table` ($columnsSql) VALUES ($placeholdersSql)";
    
    return $this->sql($query, $values); // Execute query with values
  }


  public function addX($table,$parameters){
    $keys = [];
    $values = [];
    $equals = [];
    foreach($parameters as $key => $value){
      $keys[] = $key;
      $values[] = $value;
      $equals[] = " `$key`=? ";
    }
    $equalsSql = "".implode(",", $equals)."";
    $query = "INSERT INTO `$table` SET " . $equalsSql;
    return $this->sql($query,$values);
  }


  public function tables(){
    return array_map(function($d){
      return $d["name"];
    },$this->sql("SELECT `table_name` as 'name' FROM information_schema.tables WHERE table_schema = ? ",[$this->database])); 
  }


  public function columns($table){
    return array_map(function($d){
      return $d["name"];
    },$this->sql("SELECT COLUMN_NAME as 'name' FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION",[$this->database,$table]));
  }

  public function comments($table){
    $obj = []; 
    foreach( $this->sql("SELECT COLUMN_NAME as 'name', COLUMN_COMMENT as 'comment' FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION;",[$this->database,$table]) as $row ){
      $obj[ $row["name"] ] = $row["comment"];
    }
    return $obj;
  }



  public function beginTransaction(){
    $this->db->begin_transaction();
  }

  public function autocommit(){
    $this->db->autocommit(FALSE);
  }
  
  public function commit(){
    $this->db->commit();
  }

  public function rollback(){
    $this->db->rollback();
  }

  public function save(){
    return true;
  }
}