# This script installs go on Scrutinizer for github repositories
# To use:
# "$(curl -fsSL https://raw.githubusercontent.com/phayes/go-scrutinize/master/setup.bash)" 

export PROJECT=$(echo $SCRUTINIZER_PROJECT | sed 's/g/github.com/')
export DOMAIN=$(echo $PROJECT | cut -d'/' -f1)
export OWNER=$(echo $PROJECT | cut -d'/' -f2)
export PACKAGE=$(echo $PROJECT | cut -d'/' -f3)

# Install go
wget https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz
tar -xf go1.9.linux-amd64.tar.gz
sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin

# Install bzr
sudo apt-get -y install bzr

# Set-up gopath
mkdir ~/go
export GOPATH=~/go
mkdir -p $GOPATH/src/$DOMAIN/$OWNER
ln -s ~/build $GOPATH/src/$DOMAIN/$OWNER/$PACKAGE
export PATH="$PATH:$GOPATH/bin"

# Install coverage tools
go get github.com/axw/gocov/...
go get github.com/AlekSi/gocov-xml
go get github.com/phayes/go-scrutinize
