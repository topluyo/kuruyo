/usr/bin/php \
  -d upload_max_filesize=500M \
  -d post_max_size=500M \
  -d error_log=/var/cloud/php_errors.log \
  -d log_errors=On \
  -d display_errors=On \
  -S localhost:1453 /var/cloud/server/router.php
