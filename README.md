# SCION enabled SSH
SSH client and server over SCION network, at this moment using regular quic-go connection. 
This will be replaced with SCION quic connection SOON!

## Installation steps
To install first clone and install `go-quic`:
```
mkdir -p $GOPATH/src/github.com/lucas-clemente
cd $GOPATH/src/github.com/lucas-clemente
git clone git@github.com:lucas-clemente/quic-go.git
cd quic-go
go get -t -u ./...
```

```
mkdir $GOPATH/src/github.com/xabarass
cd $GOPATH/src/github.com/xabarass
git clone git@github.com:xabarass/scion-ssh.git
cd scion-ssh
go get -t -u ./...

ssh-keygen -t rsa -f ./server/id_rsa
```

That should be it, now build server:
```
cd server
go build sshd.go
```

And client:
```
cd client
go build client.go
```

Addresses and ports are hardcoded in `client.go` and `sshd.go` files. 
