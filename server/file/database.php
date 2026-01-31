<?php 


require_once __DIR__."/index.php";

FileManager::Run_Security();



$SHOW_DATABASE = "mysql -u root --batch --skip-column-names -e \"SHOW DATABASES;\" | jq -R -s -c 'split(\"\n\")[:-1]'";


$USER_NAME = "master";
$PASSWORD  = "jn19cns0gzxmas0fjg";




if(isset($_REQUEST["action"]) && $_REQUEST["action"]=="create-database"){
  $db = $_REQUEST["name"];
  $CREATE_DATABASE = "mysql -u root -e \"CREATE DATABASE $db CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;\" ";
  echo exec($CREATE_DATABASE);
}



if(isset($_REQUEST["action"]) && $_REQUEST["action"]=="create-user"){
  $USER_NAME = $_REQUEST["name"];
  $PASSWORD  = $_REQUEST["password"];
  $CREATE_USER = "mysql -u root -e \"CREATE USER '$USER_NAME'@'localhost' IDENTIFIED BY '$PASSWORD';GRANT ALL PRIVILEGES ON *.* TO '$USER_NAME'@'localhost' WITH GRANT OPTION;FLUSH PRIVILEGES;\" ";
  echo $CREATE_USER;
  echo exec($CREATE_USER);
}



function ATOB($encoded) {
    return urldecode(base64_decode($encoded));
}

if(isset($_REQUEST["action"]) && $_REQUEST["action"]=="sql"){
  require_once __DIR__."/system/system.php";
  echo $db->sql(ATOB($_REQUEST["single"]));
  die();
}



if(isset($_REQUEST["action"]) && $_REQUEST["action"]=="sqls"){
  require_once __DIR__."/system/system.php";
  foreach( json_decode($_REQUEST["sqls"],true)  as $sql){
    echo $db->sql($sql);
  }
  die();
}



?>
<script src="//hasandelibas.github.io/documenter/documenter.js"></script>
<meta charset="utf8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<header body-class="show-menu theme-dark">
  <a href="?" title="">🗄️ Database</div></a>
  <div class="space"></div>
  <input placeholder="Search...">
</header>



# Database Create

<form grid-form action="?action=create-database" method="post">
  <label>Database Name</label>
  <input type="text" name="name">
  <label></label>
  <button>Create</button>
</form>


# Database User

<form grid-form action="?action=create-user" method="post">
  <label>User Name</label>
  <input type="text" name="name">
  <label>Password</label>
  <input type="text" name="password">
  <label></label>
  <button>Create</button>
</form>


# Database List
```
<?= exec( "mysql -u root --batch --skip-column-names -e \"SHOW DATABASES;\" | jq -R -s -c 'split(\"\n\")[:-1]'" ); ?>
```

# Database Users
```
<?php
$users_json = exec("mysql -u root --batch --skip-column-names -e \"SELECT CONCAT(User, '@', Host) FROM mysql.user;\" | jq -R -s -c 'split(\"\\n\")[:-1]'");
echo $users_json;
?>
```



# Database Import

<div flex-sx gap center>
  <button onclick="importSQL()">importSql</button>
  <button onclick="startSQL()">startSQL</button>

  
  <div id="count">0</div>
</div>

<div id="log"></div>


<script>

function ParseMultilineSql(sql, limit = Infinity) {
    const statements = [];
    let current = '';
    let inString = false;
    let stringChar = '';
    let inComment = false;

    for (let i = 0; i < sql.length; i++) {
        const char = sql[i];
        const nextChar = sql[i + 1];

        // Handle string literals (single or double quotes)
        if (!inComment && (char === '\'' || char === '"')) {
            if (!inString) {
                inString = true;
                stringChar = char;
            } else if (char === stringChar) {
                // Count preceding backslashes
                let backslashes = 0;
                let k = i - 1;
                while (k >= 0 && sql[k] === '\\') {
                    backslashes++;
                    k--;
                }
                // Close string if not escaped
                if (backslashes % 2 === 0) {
                    inString = false;
                }
            }
        }

        // Handle line comments --
        if (!inString && char === '-' && nextChar === '-') {
            inComment = true;
        }
        if (inComment && char === '\n') {
            inComment = false;
        }

        // Handle block comments /* ... */
        if (!inString && char === '/' && nextChar === '*') {
            inComment = true;
        }
        if (inComment && char === '*' && nextChar === '/') {
            inComment = false;
            i++; // Skip '/'
            current += '*/';
            continue;
        }

        current += char;

        // End of statement
        if (char === ';' && !inString && !inComment) {
            const trimmed = current.trim();
            if (trimmed.length > 0) {
                statements.push(trimmed.slice(0, -1).trim()); // remove trailing semicolon
                if (statements.length >= limit) {
                    return {
                        lines: statements,
                        sql: sql.slice(i + 1).trimStart()
                    };
                }
            }
            current = '';
        }
    }

    // Handle trailing statement without semicolon
    const leftover = current.trim();
    if (leftover && statements.length < limit) {
        statements.push(leftover);
    }

    return {
        lines: statements,
        sql: ''
    };
}







function removeLineComments(sql) {
  // Split the SQL into lines
  const lines = sql.split('\n');
  
  // Filter out lines that start with --
  const filteredLines = lines.filter(line => !line.trim().startsWith('--'));
  
  // Join back into a string
  return filteredLines.join('\n');
}


  lines = {}
  function importSQL(){
    documenter.readText().then(e=>{
      window.sql = e
      window.sql = sql.split(/^\s*--.*/m).join("")
      console.log("file readed");
      startSQL()
      //window.lines = parseSQL(sql)
      //lines.splice(0,7)
      //console.log("lines parsed");
    })
  }

  function startSQL(){
    response = ParseMultilineSql(sql,100)
    lines=response.lines;
    sql=response.sql
    sendSQL()
  }

  function BTOA(str) {
    return btoa(unescape(encodeURIComponent(str)));
  }

  function sendSQL(){
    if(lines.length==0) {
      response = ParseMultilineSql(sql,100)
      lines=response.lines;
      sql=response.sql
    }
    if(lines.length==0) {
      return
    }
    
    let line = lines[0]

    if(line.startsWith("CREATE") || line.startsWith("ALTER") || line.startsWith("INSERT")){

    }else{
      $("#log").appendChild(new Text("SKIP:["+line.substr(0,10)+"]"))
      return sendSQL()
    }
    
    documenter.post("?action=sql",{single: BTOA( line ) }).then(e=>e.text()).then(e=>{
      lines.splice(0,1)[0]
      $("#count").innerHTML = 1 + parseInt($("#count").innerHTML)
      $("#log").appendChild(new Text(e))
      sendSQL();
    })
  }
</script>