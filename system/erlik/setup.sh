/app/go build -o erlik FrontBuild.go Library.go Bash.go Sync.go LogConnection.go UnixSocketServer.go UnixSocketClient.go
sudo ln -sf "$(pwd)/erlik" /usr/local/bin/erlik