# SCION enabled SSH

SSH client and server running over SCION network. 

# Installation

## Prerequisite

SCION infrastructure has to be installed and running. Instructions can be found [here](https://github.com/scionproto/scion)

## Building the project

Clone the `scion-ssh` repository and install dependencies:

```
govendor init
govendor add +e
govendor fetch +m
```

Build the server:
```
cd $GOPATH/src/github.com/xabarass/scion-ssh/server
go build
```

Build the client
```
cd $GOPATH/src/github.com/xabarass/scion-ssh/client
go build
```

# Running

To generate server certificates:

```
cd $GOPATH/src/github.com/xabarass/scion-ssh/server

openssl req -newkey rsa:2048 -nodes -keyout key.pem -x509 -days 365 -out certificate.pem
ssh-keygen -t rsa -f ./id_rsa
```

Running the server:
```
cd $GOPATH/src/github.com/xabarass/scion-ssh/server
./server --address 1-11,[127.0.0.1]:2200
```

Running the client

```
cd $GOPATH/src/github.com/xabarass/scion-ssh/client
./client --server 1-11,[127.0.0.1]:2200 --client 1-12,[127.0.0.2]:3344
```


