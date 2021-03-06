package ssh

import(
    "fmt"
    "io/ioutil"

    "golang.org/x/crypto/ssh"

    "github.com/msteinert/pam"
)

func (s *SSHServer)PasswordAuth(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
    t, err := pam.StartFunc("", c.User(), func(s pam.Style, msg string) (string, error) {
        switch s {
        case pam.PromptEchoOff:
            return string(pass), nil
        }
        return "", fmt.Errorf("Unsupported message style")
    })
    if err != nil {
        return nil, err
    }
    err = t.Authenticate(0)
    if err != nil {
        return nil, fmt.Errorf("Authenticate: %s", err.Error())
    }
    
    return nil, nil
}

func loadAuthorizedKeys(file string)(map[string]bool, error){
    authKeys := make(map[string]bool)

    authorizedKeysBytes, err := ioutil.ReadFile(file)
    if err != nil {
        return nil, err
    }

    for len(authorizedKeysBytes) > 0 {
        pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
        if err != nil {
            return nil, err
        }

        authKeys[string(pubKey.Marshal())] = true
        authorizedKeysBytes = rest
    }

    return authKeys, nil
}

func (s *SSHServer)PublicKeyAuth(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
    authKeys, err := loadAuthorizedKeys(s.authorizedKeysFile)
    if err != nil {
        return nil, fmt.Errorf("Failed loading authorized files: %v", err)
    }

    if authKeys[string(pubKey.Marshal())] {
        return &ssh.Permissions{
            // Record the public key used for authentication.
            Extensions: map[string]string{
                "pubkey-fp": ssh.FingerprintSHA256(pubKey),
            },
        }, nil
    }

    return nil, fmt.Errorf("Unknown public key for %q", c.User())
}