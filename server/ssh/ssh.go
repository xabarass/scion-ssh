package ssh

import(
    "fmt"
    "net"
    "log"
    "io/ioutil"

    "golang.org/x/crypto/ssh"

    "github.com/xabarass/scion-ssh/server/config"
)

type ChannelHandlerFunction func(newChannel ssh.NewChannel)

type SSHServer struct {
    authorizedKeysFile string

    configuration *ssh.ServerConfig

    channelHandlers map[string]ChannelHandlerFunction
}

func Create(config *config.ServerConfig, version string)(*SSHServer, error){
    server := &SSHServer{
        authorizedKeysFile:config.AuthorizedKeysFile,
        channelHandlers:make(map[string]ChannelHandlerFunction),
    }

    server.configuration=&ssh.ServerConfig{
        PasswordCallback:server.PasswordAuth,
        PublicKeyCallback:server.PublicKeyAuth,
        NoClientAuth : config.AllowNoAuth,
        MaxAuthTries: 3,
        // ServerVersion: fmt.Sprintf("SCION-ssh-server-v%s", version),
    }

    privateBytes, err := ioutil.ReadFile(config.SSHKeyPath)
    if err != nil {
        return nil, fmt.Errorf("Failed loading private key: %v", err)
    }
    private, err := ssh.ParsePrivateKey(privateBytes)
    if err != nil {
        return nil, fmt.Errorf("Failed parsing private key: %v", err)
    }
    server.configuration.AddHostKey(private)

    server.channelHandlers["session"]=handleSession

    return server, nil
}

func (s *SSHServer)handleChannels(chans <-chan ssh.NewChannel) {
    // Service the incoming Channel channel in go routine
    for newChannel := range chans {
        go s.handleChannel(newChannel)
    }
}

func (s *SSHServer)handleChannel(newChannel ssh.NewChannel){
    if handler, exists := s.channelHandlers[newChannel.ChannelType()]; exists{
        handler(newChannel)
    }else{
        newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", newChannel.ChannelType()))
        return
    }
}

func (s *SSHServer)HandleConnection(conn net.Conn)(error){
    log.Printf("Handling new connection")
    sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.configuration)
    if err != nil {
        log.Printf("Failed to create new connection (%s)", err)
        conn.Close()
        return err
    }

    log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
    // Discard all global out-of-band Requests
    go ssh.DiscardRequests(reqs)
    // Accept all channels
    s.handleChannels(chans)

    return nil
}
