<script src="//hasandelibas.github.io/documenter/documenter.js"></script>
<meta charset="utf8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">

<header body-class="show-menu theme-dark tab-system">
  <div title="" hover onclick="location.reload()">
    💼 CronJob
  </div>
  <div class="space"></div>
  <script> documenter.on("ready",()=>document.body.setAttribute("spellcheck","false") ) </script>
</header>


<style>
  html{
    position:relative;
  }
  html:before{
    content:"";
    position:absolute;
    left:0;right:0;
    bottom:0;top:0;
    width:100%;
    height:100%;
    z-index:-1;
    background: url(https://source.unsplash.com/random/1200x800/?space);
    background-size: cover;
    opacity:.7;
  }
</style>

# JOBS



<?php 
  if(@$_GET['action']=="create-job"){
    $command = $_POST["code"];
    echo "```\n";
    echo "# Creating JOB \n";
    passthru($command);
    echo "```\n";
  }
?>

## list
```
<?= passthru("crontab -l") ?>
```



## delete all
<?php 
if(@$_GET['action']=="delete-jobs"){
  echo "```\n";
  passthru("crontab -r");
  echo "```\n";
}
?>
<a button style="background:red;" href="?action=delete-jobs"> crontab -r [Hepsini sil] </a>


## new 
```
* * * * * command_to_run
│ │ │ │ │
│ │ │ │ └─ Day of the week (0-7) (Sunday = 0 or 7)
│ │ │ └──── Month (1 - 12)
│ │ └─────── Day of the month (1 - 31)
│ └───────── Hour (0 - 23)
└─────────── Minute (0 - 59)
```

```
* — Expands to all values for the field
, — List separator
- — Range separator
/ — Step values (e.g., */5 for every 5 units)
```



<label for="cron-select">Select Schedule:</label><br>
<select id="cron-select" onchange="updateCronLine()" style="width: 100%;">
  <option value="* * * * *">Every minute</option>
  <option value="*/5 * * * *">Every 5 minutes</option>
  <option value="0 * * * *">Every hour</option>
  <option value="0 */2 * * *">Every 2 hours</option>
  <option value="0 0 * * *">Every day (midnight)</option>
  <option value="0 6 * * *">Every day at 6 AM</option>
  <option value="0 12 * * *">Every day at 12 PM (noon)</option>
  <option value="0 18 * * *">Every day at 6 PM</option>
  <option value="0 0 * * 0">Every Sunday</option>
  <option value="0 0 * * 1-5">Every weekday (Mon–Fri)</option>
  <option value="0 0 1 * *">First day of the month</option>
  <option value="0 0 15 * *">15th of every month</option>
  <option value="0 0 1 1 *">January 1st (once a year)</option>
</select>

<br><br>

<label for="cron-command">Cron Job Command:</label><br>
<input type="text" id="cron-command" value="/web/jobs/every-minute.sh" style="width: 100%;" oninput="updateCronLine()">

<br><br>

<form id="cron-form" method="post" action="?action=create-job" onsubmit="submitCron(event)">
  <label for="create-cron-job-textarea">Cron Job Script:</label><br>
  <textarea id="create-cron-job-textarea" name="code" rows="10" cols="60" style="width: 100%; height: 160px;">
( crontab -l 2>/dev/null; echo "0 2 * * * /usr/local/bin/backup.sh" ) | crontab -
  </textarea><br><br>

  <button type="submit">Create Cron Job</button>
</form>

<script>
function updateCronLine() {
  const schedule = document.getElementById('cron-select').value;
  const command = document.getElementById('cron-command').value.trim();
  const textarea = document.getElementById('create-cron-job-textarea');
  textarea.value = `( crontab -l 2>/dev/null; echo "${schedule} ${command}" ) | crontab -`
  //textarea.value = `echo "${schedule} root ${command}" | sudo tee /etc/cron.d/backup-cron >/dev/null`
}
document.addEventListener("DOMContentLoaded",function(){
  updateCronLine();
})
</script>


