package main

import (
	"log"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/scionproto/scion/go/scion-ssh/scionutils"
	"github.com/scionproto/scion/go/lib/snet"
	"github.com/scionproto/scion/go/lib/snet/squic"
	"github.com/scionproto/scion/go/scion-ssh/sshserver/config"
	"github.com/scionproto/scion/go/scion-ssh/sshserver/ssh"
	"github.com/scionproto/scion/go/scion-ssh/quicconn"
)

const (
	VERSION = "1.0"
)

var (
	// Connection
	LISTEN_ADDRESS = kingpin.Flag("address", "SCION address to listen on").Required().String()

	// Configuration file
	CONFIGRATION_FILE = kingpin.Flag("config", "SSH server configuration file").Default("config.toml").ExistingFile()
)

func initSCIONConnection(serverAddress, tlsCertFile, tlsKeyFile string)(*snet.Addr, error){
	log.Println("Initializing SCION connection")

	serverCCAddr, err := snet.AddrFromString(serverAddress)
	if err != nil {
		return nil, err
	}

	err = snet.Init(serverCCAddr.IA, scionutils.GetSciondAddr(serverCCAddr), scionutils.GetDispatcherAddr(serverCCAddr))
	if err != nil {
		return serverCCAddr, err
	}

	err = squic.Init(tlsKeyFile, tlsCertFile)
	if err != nil {
		return serverCCAddr, err
	}

	return serverCCAddr, nil
}

func main() {
	log.Println("Starting SCION SSH server...")
	kingpin.Parse()

	conf, err := config.Load(*CONFIGRATION_FILE)
	if err != nil {
		log.Panicf("Error loading configuration: %s", err)
	}

	serverCCAddr, err := initSCIONConnection(*LISTEN_ADDRESS, conf.Connection.QuicTLSCert, conf.Connection.QuicTLSKey)
	if err != nil {
		log.Panicf("Error initializing SCION connection: %s", err)
	}

	sshServer, err := ssh.Create(&conf.Server, VERSION)
	if err != nil {
		log.Panicf("Error creating ssh server: %s", err)
	}

	listener, err := squic.ListenSCION(nil, serverCCAddr)
	if err != nil {
		log.Fatalf("Failed to listen (%s)", err)
	}

	log.Printf("Starting to wait for connections")
	for {
		//TODO: Check when to close the connections
		sess, err := listener.Accept()
	    if err != nil {
	    	log.Printf("Failed to accept session", err)
	    	continue
	    }
	    stream, err := sess.AcceptStream()
		if err != nil {
			log.Printf("Failed to accept incoming connection (%s)", err)
			continue
		}
		
	    qc := &quicconn.QuicConn{Session:sess, Stream:stream}

	    sshServer.HandleConnection(qc)
	}
}
