<?php 




require_once __DIR__."/../index.php";
FileManager::Run_Security();

$user = [
  "user_id" => 3,
  "user_name" => "TopluyoMod",
  "user_nick" => "TopluyoMod"
];


require_once "/var/web/system/system.php";




function idfy($input,$key="id"){
  $output = [];
  foreach ($input as $item) {
    $output[$item[$key]] = $item;
  }
  return $output;
}

function groupify($input,$key="id"){
  $output = [];
  foreach ($input as $item) {
    $output[$item[$key]][] = $item;
  }
  return $output;
}

function trim_special($text, $char) {
  return rtrim(ltrim($text, $char), $char);
}

function urlfiy($u){
  return str_replace("//","/",$u);
}
function base(...$url){
  $url = join("/",$url);
  $url = trim_special($url,"/");
  return "https://topluyo.com".urlfiy("/".$url );
}


function split($str, $function = null) {
  if (!is_string($str)) {
    return [];
  }

  $arr = array_filter(array_map('trim', explode(',', $str)), function ($value) {
    return $value !== '';
  });
  
  if ($function !== null && is_callable($function)) {
    return array_map($function, $arr);
  }
  return $arr;
}






?>

<script src="//hasandelibas.github.io/documenter/documenter.js"></script>
<meta charset="utf8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<header body-class="show-menu theme-dark tab-system">

  <script>
    documenter.when("[documenter-icon]",function(el){
      if(documenter.icons[el.getAttribute("documenter-icon")]!=null){
        el.innerHTML = documenter.icons[el.getAttribute("documenter-icon")].split("<svg").join('<svg fill="currentColor" style="width:1.5em;height:1.5em" ');
      }else{
        el.innerHTML = el.attr("documenter-icon")
      }
    })
  </script>

  <div title="">📄 mod.topluyo.com</div>
  
  <div class="space"></div>
  <div flex-x center gap>
    <img src='<?= $user['user_image'] ?>' style="width:42px;height:42px;object-fit:cover;border-radius:100%;">
    <div flex-y>
      <span><?= $user['user_name'] ?></span>
      <span style="font-size:0.8em;opacity:0.8;">@<?= $user['user_nick'] ?></span>
    </div>
  </div>

  <div onclick='document.cookie.split(";").forEach(function(c){var d=c.indexOf("="),n=d>-1?c.substr(0,d).trim():c.trim();document.cookie=n+"=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/"});' documenter-icon="power" icon-button></div>

</header>

# Rozet Ver
<?php 

if(get("action")=="add-badge"){
  $_user = $db->get("user",["nick"=>all("nick")]);
  if($_user){
    $badges = split($_user["badges"]);
    $badges[] = all("badge_id");
    $badges = array_unique( $badges );
    $badges = join(",",$badges);
    $db->def("user",["id"=>$_user['id'],"badges"=>$badges]);

    file_put_contents("log.txt", $user["user_nick"]." kullanıcısı -> ". $_user['nick'] . " kullanıcısına -> " . all("badge_id") . " rozetini verdi\n", FILE_APPEND | LOCK_EX);
  }
}

?>
<form grid-form method="post" action="?action=add-badge">
  <label>Kullanıcı Adı:</label>
  <input name="nick" />
  <label>Verilecek Badge</label>
  <select name="badge_id">
    <?php foreach($db->all("badge") as $badge){ ?>
      <option value="<?= $badge['id'] ?>"><?= $badge['name'] ?></option>
    <?php } ?>
  </select>
  <label></label>
  <button>Ekle</button>
</form>




# Topluyodan Engelle
<?php 

if(get("action")=="block-tp"){
  $_user = $db->get("user",["nick"=>all("nick")]);
  if($_user){
    $db->def("user",["id"=>$_user['id'],"blocked"=>1,"blocked_reason"=>all("reason")]);
    file_put_contents("log.txt", $user["user_nick"]." kullanıcısı -> ". $_user['nick'] . " kullanıcısını -> " . all("reason") . " sebebiyle engelledi\n", FILE_APPEND | LOCK_EX);
  }
}

?>
<form grid-form method="post" action="?action=block-tp">
  <label>Kullanıcı Adı:</label>
  <input name="nick" />
  <label>Sebep</label>
  <input name="reason">
  <label></label>
  <button>Engelle</button>
</form>





# Engelli Kaldır
<?php 
if(get("action")=="unblock-tp"){
  $_user = $db->get("user",["nick"=>all("nick")]);
  if($_user){
    $db->def("user",["id"=>$_user['id'],"blocked"=>0,"blocked_reason"=>""]);
    file_put_contents("log.txt", $user["user_nick"]." kullanıcısı -> ". $_user['nick'] . " kullanıcısını -> " . all("reason") . " sebebiyle engeli kaldırdı\n", FILE_APPEND | LOCK_EX);
  }
}
?>
<form grid-form method="post" action="?action=unblock-tp">
  <label>Kullanıcı Adı:</label>
  <input name="nick" />
  <label></label>
  <button>Engelle</button>
</form>




# Sunucuyu Kapat
<?php 

if(get("action")=="block-tp"){
  $group = $db->get("group",["nick"=>all("nick")]);
  if($group){
    $db->def("group",["id"=>$group['id'],"blocked"=>1,"home"=>"# <div style='color:red'>Bu Sunucu Kapatılmıştır</div>\n".all("reason")]);
    file_put_contents("log.txt", $user["user_nick"]." kullanıcısı -> ". $group['nick'] . " sunucusunu -> " . all("reason") . " sebebiyle kapattı\n", FILE_APPEND | LOCK_EX);
  }
}

?>
<form grid-form method="post" action="?action=block-tp">
  <label>Sunucu Adı:</label>
  <input name="nick" />
  <label>Sebep</label>
  <input name="reason">
  <label></label>
  <button>Sunucuyu Kapat</button>
</form>


# Sunucu Seviyesi Ver
<?php 

if(get("action")=="set-level"){
  $_group = $db->get("group",["nick"=>all("nick")]);
  if($_group){
    $db->def("group",["id"=>$_group['id'],"level"=>all("level")]);

    file_put_contents("log.txt", $user["user_nick"]." kullanıcısı -> ". $_group['nick'] . " sunucusuna -> " . all("level") . "seviyesini verdi\n" , FILE_APPEND | LOCK_EX);
  }
}

?>
<form grid-form method="post" action="?action=set-level">
  <label>Sunucu Nick'i:</label>
  <input name="nick" />
  <label>Seviye</label>
  <input name="level" type="number" />
  <label></label>
  <button>Ata</button>
</form>




# Rozetliler
<?php 
$organizations = $db->sql("SELECT 
  `badge`.`id` as 'badge_id',
  `badge`.`image` as 'badge_image',
  `badge`.`name` as 'badge_name',
  `user`.`name` as 'user_name',
  `user`.`nick` as 'user_nick',
  `user`.`image` as 'user_image'
FROM `badge`
LEFT JOIN `user` ON FIND_IN_SET(`badge`.`id`, `user`.`badges` )
");


$organizations = groupify($organizations,"badge_id");

$titles = $db->all("badge");
$titles = idfy($titles);

?>

<div box view="organization" style="overflow-y:auto;">
  <div>
    <?php foreach($titles as $title){ ?>
      <div flex-x center gap>
        <?php if($title["image"]){ ?>
          <img style="width:1em;height:1em;" src="<?=  $title["image"]  ?>" >
        <?php } ?>
        <?= $title["name"] ?>
      </div>
      
      <?php if(array_key_exists($title['id'],$organizations)){ ?>
        <div flex-x gap style="flex-flow: wrap;">
          <?php foreach($organizations[$title['id']] as $organization){ ?>
            <a underline user-nick="<?= $organization["user_nick"] ?>" in-content flex-x center gap href="<?= base("@".$organization['user_nick']) ?>">
              <img style="width:2em;height:2em;" round src="<?=  $organization['user_image']  ?>">
              <div organization-user-name><?=  $organization['user_name']  ?></div>
            </a>
          <?php } ?>
        </div>
      <?php }else{ ?>
        <div flex mute>Bu ünvana sahip üye yok</div>
      <?php } ?>
    <?php } ?>
  </div>  
</div>






# Uygulama Revizyonları
<?php 

if(all("app-reject")){
  $reason = all("reason");
  $db->def("app_revision",["id"=>all("app-reject"),"reason"=>$reason,"request"=>0]);
}

if(all("app-verify")){
  $app = $db->get("app_revision",all("app-verify"));
  if($app!=false) {
    $db->def("app_revision",["id"=> all("app-verify"),"verified"=>1,"request"=>0]);
    $db->def("app",[
      "id"           => $app["app_id"],
      "name"         => $app["name"],
      "description"  => $app["description"],
      "image"        => $app['image'],
      "icon"         => $app["icon"],
      "app_type_id"  => $app["app_type_id"],
      "link"         => $app["link"],
      "css"          => $app["css"],
      "webhook"      => $app["webhook"],
      "verify"       => 1
    ]);
  }
}

?>

<?php foreach($db->sql("SELECT * FROM app_revision WHERE request=1 ") as $app){ ?>
  <table>
    <tr><th colspan=2> Uygulama ( <?= $db->get("app_type",$app['app_type_id'])['name'] ?> ) </th></tr>
    <?php foreach($app as $key=>$value) { ?>
      <tr>
        <th style="font-weight:bold;"><?= $key ?></th>
        <td><?= ($key=="image" || $key=="icon") ? "<img src='$value'>" : $value ?></td>
      </tr>
    <?php } ?>

    <tr><th colspan=2> Yayıncı </th></tr>
    <?php foreach($db->sql("SELECT id, nick, image, name, created_at FROM `user` WHERE id=?", [$db->get('app',$app['id'])['user_id'] ] )[0] as $key=>$value) { ?>
      <tr>
        <th style="font-weight:bold;"><?= $key ?></th>
        <td><?= ($key=="image" || $key=="icon") ? "<img src='$value'>" : $value ?></td>
      </tr>
    <?php } ?>

    <tr>
      <td>Onayla</td>
      <td>
        <form action="?" method="post">
          <input style="display:none" name="app-verify" value="<?= $app['id'] ?>">
          <button style="background:green;">Onayla</button>
        </form>
      </td>
    </tr>

    <tr>
      <td>Reddet</td>
      <td>
        <form action="?" method="post">
          <input style="display:none" name="app-reject" value="<?= $app['id'] ?>">
          <input style="width:140px" name="reason" placeholder="sebep">
          <button style="background:red;">Reddet</button>
        </form>
      </td>
    </tr>

  </table>
<?php } ?>


















# Silinecek Marketteki Uygulamalar
<?php 

if(post("app-delete") && post("delete")=="delete"){
  $db->remove("app",["id"=>post("app-delete")]);
}

?>
<table>
<?php foreach($db->all("app",['verify'=>0, "waiting_verify"=>1]) as $app){ ?>
<tr>
  <td><?= $app['name'] ?></td>
  <td><a target="_blank" href="?revision-verify=<?= $app['id'] ?>">onayla</a></td>
  <td><a target="_blank" href="https://alfa.topluyo.com/~market/app/<?=  $app['id'] ?>">incele</a></td>
  <td> 
    <form action="?" method="post">
      <input style="width:140px" name="delete" placeholder="delete">
      <input style="display:none" name="app-delete" value="<?= $app['id'] ?>">
      <button style="background:red;">Kaldır</button>
    </form>
  
  </td>
</tr>
<?php } ?>
</table>