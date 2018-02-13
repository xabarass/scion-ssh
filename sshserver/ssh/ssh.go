package ssh

import(
    "fmt"
    "io/ioutil"
    "net"
    "log"

    "golang.org/x/crypto/ssh"

    "github.com/xabarass/scion-ssh/sshserver/config"
)

type ChannelHandlerFunction func(newChannel ssh.NewChannel)

type SSHServer struct {
    authorizedKeys map[string]bool

    configuration *ssh.ServerConfig

    channelHandlers map[string]ChannelHandlerFunction
}

func Create(config *config.ServerConfig, version string)(*SSHServer, error){
    server := &SSHServer{
        authorizedKeys:make(map[string]bool),
        channelHandlers:make(map[string]ChannelHandlerFunction),
    }

    err := loadAuthorizedKeys(config.AuthorizedKeysFile, server)
    if err != nil {
        return nil, fmt.Errorf("Failed loading authorized files: %v", err)
    }

    server.configuration=&ssh.ServerConfig{
        PasswordCallback:server.PasswordAuth,
        PublicKeyCallback:server.PublicKeyAuth,
        NoClientAuth : config.AllowNoAuth,
        MaxAuthTries: config.MaxAuthTries,
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

func loadAuthorizedKeys(file string, server *SSHServer)(error){
    authorizedKeysBytes, err := ioutil.ReadFile(file)
    if err != nil {
        return err
    }

    for len(authorizedKeysBytes) > 0 {
        pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
        if err != nil {
            return err
        }

        server.authorizedKeys[string(pubKey.Marshal())] = true
        authorizedKeysBytes = rest
    }

    return nil
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
    sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.configuration)
    if err != nil {
        log.Printf("Failed to create new connection (%s)", err)
        return err
    }

    log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
    // Discard all global out-of-band Requests
    go ssh.DiscardRequests(reqs)
    // Accept all channels
    go s.handleChannels(chans)

    return nil
}