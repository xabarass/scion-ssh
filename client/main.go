package main

import (
    "log"
    "net"
    "fmt"
    "path"
    "os/user"

    "golang.org/x/crypto/ssh/terminal"

    "gopkg.in/alecthomas/kingpin.v2"
    
    "github.com/scionproto/scion/go/lib/snet/squic"
    "github.com/scionproto/scion/go/lib/snet"

    "github.com/xabarass/scion-ssh/client/ssh"
    "github.com/xabarass/scion-ssh/quicconn"
    "github.com/xabarass/scion-ssh/scionutils"
)

const (
    VERSION = "1.0"
)

var (
    // Connection
    SERVER_ADDRESS = kingpin.Flag("server", "SSH server's SCION address").Required().String()
    CLIENT_ADDRESS = kingpin.Flag("client", "client's SCION address").Required().String()

    //TODO: additional file paths
    KNOWN_HOSTS_FILE = kingpin.Flag("known_hosts", "File where known hosts are stored").Default("known_hosts").String()
    IDENTITY_FILE = kingpin.Flag("identity", "Identity (private key) file").String()

    USER = kingpin.Flag("user", "Username to authenticate with").String()
)

func initSCIONConnection(serverAddress, clientAddress string)(*snet.Addr, *snet.Addr, error){
    log.Println("Initializing SCION connection")

    serverCCAddr, err := snet.AddrFromString(serverAddress)
    if err != nil {
        return nil, nil, err
    }
    clientCCAddr, err := snet.AddrFromString(clientAddress)
    if err != nil {
        return nil, nil, err
    }

    err = snet.Init(clientCCAddr.IA, scionutils.GetSciondAddr(clientCCAddr), scionutils.GetDispatcherAddr(clientCCAddr))
    if err != nil {
        return nil, nil, err
    }

    return serverCCAddr, clientCCAddr, nil
}

func PromptPassword() (secret string, err error){
    fmt.Printf("Password: ")
    password, _ := terminal.ReadPassword(0)
    fmt.Println()
    return string(password), nil
}

func PromptAcceptHostKey(hostname string, remote net.Addr, publicKey string)(bool){
    fmt.Printf("Key fingerprint MD5 is: %s do you recognize it? (yes/no) ", publicKey)
    var answer string
    fmt.Scanln(&answer)    
    if(answer=="yes"){
        return true    
    }else{
        return false
    }
    
}

func main(){
    kingpin.Parse()

    curentUser:="nobody"
    privateKeyFile:="id_rsa"
    if u, err := user.Current(); err==nil{
        curentUser=u.Username
        privateKeyFile=path.Join(u.HomeDir,".ssh", "id_rsa")
    }
    if(*USER!=""){
        curentUser=*USER   
    }
    if(*IDENTITY_FILE!=""){
        privateKeyFile=*IDENTITY_FILE
    }

    // Initialize SCION library
    serverCCAddr, clientCCAddr, err := initSCIONConnection(*SERVER_ADDRESS, *CLIENT_ADDRESS)
    if err != nil {
        log.Panicf("Error initializing SCION connection: %s", err)
    }

    // Establish connection with remote server
    sess, err := squic.DialSCION(nil, clientCCAddr, serverCCAddr)
    if err != nil {
        log.Panicf("Error dialing SCION! %s", err)
    }
    stream, err := sess.OpenStreamSync()
    if err != nil {
        log.Panicf("Error opening stream! %s", err)
    }
    qc := &quicconn.QuicConn{Session:sess, Stream:stream}

    // Create SSH client
    sshConfig := &ssh.SSHClientConfig{
        VerifyHostKey:true,
        VerifyNewKeyHandler:PromptAcceptHostKey,
        KnownHostKeyFile:*KNOWN_HOSTS_FILE,

        UsePasswordAuth:true,
        PassAuthHandler: PromptPassword,

        UsePublicKeyAuth: true,
        PrivateKeyPath: privateKeyFile,
    }

    sshClient, err := ssh.Create(curentUser, VERSION, sshConfig)
    if err!= nil{
        log.Panicf("Error creating ssh client %s", err)
    }

    err=sshClient.Connect(qc)
    if err!= nil{
        log.Panicf("Error connecting %s", err)
    }
    defer sshClient.Close()

    sshClient.Shell()
}