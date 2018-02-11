package ssh

import(
    "fmt"

    "golang.org/x/crypto/ssh"
)

func (s *SSHServer)PasswordAuth(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
    //TODO: Implement me!
    // return nil, fmt.Errorf("Password authentication not yet implemented!")
    return nil, nil
}

func (s *SSHServer)PublicKeyAuth(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
    if s.authorizedKeys[string(pubKey.Marshal())] {
        return &ssh.Permissions{
            // Record the public key used for authentication.
            Extensions: map[string]string{
                "pubkey-fp": ssh.FingerprintSHA256(pubKey),
            },
        }, nil
    }

    return nil, fmt.Errorf("Unknown public key for %q", c.User())
}