package ssh

import(
    "fmt"    
    "net"
    "log"
    "os"
    "io/ioutil"
    "crypto/md5"

    "golang.org/x/crypto/ssh"

    // knownhosts doesn't support SCION address format, so i had to change it
    "github.com/xabarass/scion-ssh/client/ssh/knownhosts"
)

type AuthenticationHandler func() (secret string, err error)
type VerifyHostKeyHandler func(hostname string, remote net.Addr, key string)(bool)

type SSHClientConfig struct {
    // Host key verification
    VerifyHostKey bool
    KnownHostKeyFile string
    VerifyNewKeyHandler VerifyHostKeyHandler

    UsePasswordAuth bool
    PassAuthHandler AuthenticationHandler

    UsePublicKeyAuth bool
    PrivateKeyPath string
}

type SSHClient struct {
    config ssh.ClientConfig
    promptForForeignKeyConfirmation VerifyHostKeyHandler
    knownHostsFileHandler ssh.HostKeyCallback
    knownHostsFilePath string

    session *ssh.Session
    //Known host keys
}

func Create(username, version string, config *SSHClientConfig) (*SSHClient, error){
    client:=&SSHClient{
        config:ssh.ClientConfig{
            User:          username,
            // ClientVersion: fmt.Sprintf("SCION-SSH-%s", version),
        },
    }

    var authMethods []ssh.AuthMethod

    // Load client private key
    if config.UsePublicKeyAuth {
        am, err := loadPrivateKey(config.PrivateKeyPath)
        if err != nil {
            log.Printf("Error loading private key, skipping authentication step")
        }else{
            authMethods=append(authMethods, am)    
        }
    }

    // Use password auth
    if config.UsePasswordAuth {
        log.Printf("Configuring password auth")
        authMethods=append(authMethods, ssh.PasswordCallback(config.PassAuthHandler))
    }

    if config.VerifyHostKey {
        // Create file if doesn't exist
        if _, err := os.Stat(config.KnownHostKeyFile); os.IsNotExist(err){
            var file, err = os.Create(config.KnownHostKeyFile)
            if err!=nil{
                return nil, err
            }
            file.Close()
        }

        client.knownHostsFilePath=config.KnownHostKeyFile
        khh, err := knownhosts.New(config.KnownHostKeyFile)
        if err!=nil{
            return nil, err
        }
        client.knownHostsFileHandler=khh
        client.config.HostKeyCallback = client.verifyHostKey
        client.promptForForeignKeyConfirmation=config.VerifyNewKeyHandler
    }else{
        log.Printf("Not verifying host key!")
        client.config.HostKeyCallback=ssh.InsecureIgnoreHostKey()
    }
    client.config.Auth=authMethods

    return client, nil
}

func (client *SSHClient)Connect(transportStream net.Conn)(error){
    c, nc, rc, err := ssh.NewClientConn(transportStream, transportStream.RemoteAddr().String(), &client.config)
    if err != nil {
        return err
    }
    conn := ssh.NewClient(c, nc, rc)

    client.session, err = conn.NewSession()
    if err != nil {
        return err
    }

    return nil
}

func (c *SSHClient)Close(){
    c.session.Close()
}

func loadPrivateKey(filePath string)(ssh.AuthMethod, error){
    key, err := ioutil.ReadFile(filePath)
    if err != nil {
        return nil, err
    }

    privateKey, err := ssh.ParsePrivateKey(key)
    if err != nil {
        return nil, err
    }

    return ssh.PublicKeys(privateKey), nil
}

func (c *SSHClient)verifyHostKey(hostname string, remote net.Addr, key ssh.PublicKey) error{
    log.Printf("Checking new host signature host: %s", remote.String())

    err := c.knownHostsFileHandler(hostname, remote, key)
    if err!=nil{
        switch e := err.(type) {
            case *knownhosts.KeyError:
                if (len(e.Want)==0){
                    // It's an unknown key, prompt user!
                    hash := md5.New()
                    hash.Write(key.Marshal())
                    if (c.promptForForeignKeyConfirmation(hostname, remote, fmt.Sprintf("%x", hash.Sum(nil)))){
                        newLine:=knownhosts.Line([]string{remote.String()}, key)
                        err=appendFile(c.knownHostsFilePath, newLine)
                        if err!=nil{
                            fmt.Printf("Error appending line to known_hosts file %s", err)
                        }
                        return nil
                    }else{
                        return fmt.Errorf("Unknown remote host's public key!")    
                    }
                }else{
                    // Host's signature has changed, error!
                    return err
                }
            default:
                // Unknown error
                return err
        }

    }else{
        return nil
    }
}

func appendFile(fileName, text string)(error){
    f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0600)
    if err != nil {
        return err
    }
    defer f.Close()

    if _, err = f.WriteString(text); err != nil {
        return err
    }
    f.WriteString("\n");

    return nil
}
