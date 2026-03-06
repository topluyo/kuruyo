echo "==========================================="
echo "==              GOLANG                  ==="
echo "==========================================="

apt update
apt install -y git

rm -rf /usr/local/go
rm -rf /usr/bin/go
apt remove -y golang-go 



GO_VERSION="1.22.6"

wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
# Add to ~/.profile ONLY if not already added
PROFILE="$HOME/.profile"

grep -qxF 'export PATH=$PATH:/usr/local/go/bin' "$PROFILE" || \
    echo 'export PATH=$PATH:/usr/local/go/bin' >> "$PROFILE"
grep -qxF 'export GOPATH=$HOME/go' "$PROFILE" || \
    echo 'export GOPATH=$HOME/go' >> "$PROFILE"
grep -qxF 'export PATH=$PATH:$GOPATH/bin' "$PROFILE" || \
    echo 'export PATH=$PATH:$GOPATH/bin' >> "$PROFILE"
source "$PROFILE"
source ~/.profile
go version

rm go${GO_VERSION}*.tar.gz

echo "======================="
echo "please write this command:"
echo "source ~/.profile"
echo "PLEASE REBOOT"
