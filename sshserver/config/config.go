package config

import(
    "github.com/BurntSushi/toml"
)

type ConnectionConfig struct {
    QuicTLSCert string  `toml:"tls_cert"`
    QuicTLSKey string   `toml:"tls_key"`
}

type ServerConfig struct {
    SSHKeyPath string   `toml:"ssh_key_path"`

    AllowNoAuth bool    `toml:"no_client_auth"`
    AllowPasswordAuth bool    `toml:"password_auth"`
    MaxAuthTries int    `toml:"max_auth_tries"`

    AuthorizedKeysFile string   `toml:"authorized_keys_file"`
}

type Config struct {
    Connection ConnectionConfig
    Server ServerConfig
}

func Load(configFile string)(*Config, error){
    var conf Config

    _, err := toml.DecodeFile(configFile, &conf)
    return &conf, err
}