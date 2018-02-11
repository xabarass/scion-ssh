package main

import (
    "fmt"
    "os"
    "github.com/docker/docker/pkg/term"
    "strings"
    "github.com/nanobox-io/golang-ssh"
    "os/signal"
    "syscall"
    "encoding/binary"
    "strconv"
    "log"

    mainssh "golang.org/x/crypto/ssh"
    // quic "github.com/lucas-clemente/quic-go"
    "github.com/scionproto/scion/go/lib/snet/squic"
    "github.com/scionproto/scion/go/lib/snet"
    "github.com/scionproto/scion/go/scion-ssh/myconn"
)

// For now let it be hardcoded
const (
    USERNAME="scion"
    PASSWORD="supersecure"
    ADDRESS="127.0.0.1"
    PORT=2200
)

func main() {
    if(len(os.Args)!=3){
        log.Fatal("Not enough parameters, you must specfy server_address and client_address")
    }

    serverAddress := os.Args[1]
    clientAddr := os.Args[2]

    err := connect(serverAddress, clientAddr)
    if err != nil {
        fmt.Printf("Failed to connect - %s\n", err)
    }
}

func connect(serverAddr, clientAddr string) error {
    nanPass := ssh.Auth{Passwords: []string{PASSWORD}}
    client, err := ssh.NewNativeClient(USERNAME, ADDRESS, "SSH-2.0-CustomClient-1.0", PORT, &nanPass)
    if err != nil {
        return fmt.Errorf("Failed to create new client - %s", err)
    }
    
    err = shell(client, serverAddr, clientAddr)
    if err != nil && err.Error() != "exit status 255" {
        return fmt.Errorf("Failed to request shell - %s", err)
    }

    return nil
}

func shell(client *ssh.NativeClient, serverAddr, clientAddr string, args ...string) error {
    var (
        termWidth, termHeight = 80, 24
    )

    serverCCAddr, _ := snet.AddrFromString(serverAddr)
    clientCCAddr, _ := snet.AddrFromString(clientAddr)

    sciondAddr := "/run/shm/sciond/sd" + strconv.Itoa(clientCCAddr.IA.I) + "-" + strconv.Itoa(clientCCAddr.IA.A) + ".sock"
    dispatcherAddr := "/run/shm/dispatcher/default.sock"
    snet.Init(clientCCAddr.IA, sciondAddr, dispatcherAddr)
    squic.Init("","")

    // conn, err := mainssh.Dial("tcp", fmt.Sprintf("%s:%d", client.Hostname, client.Port), &client.Config)
    addr:=fmt.Sprintf("%s:%d", client.Hostname, client.Port)
    sess, err := squic.DialSCION(nil, clientCCAddr, serverCCAddr)
    // sess, err := quic.DialAddr(addr, &tls.Config{InsecureSkipVerify: true}, nil)
    if err != nil {
        return err
    }
    stream, err := sess.OpenStreamSync()
    if err != nil {
        return err
    }
    mc := &myconn.MyConn{Session:sess, Stream:stream}
    c, nc, rc, err := mainssh.NewClientConn(mc, addr, &client.Config)
    if err != nil {
        return err
    }
    conn := mainssh.NewClient(c, nc, rc)

    session, err := conn.NewSession()
    if err != nil {
        return err
    }

    defer session.Close()

    session.Stdout = os.Stdout
    session.Stderr = os.Stderr
    session.Stdin = os.Stdin

    modes := mainssh.TerminalModes{
        mainssh.ECHO: 1,
    }

    fd := os.Stdin.Fd()

    if term.IsTerminal(fd) {
        oldState, err := term.MakeRaw(fd)
        if err != nil {
            return err
        }

        defer term.RestoreTerminal(fd, oldState)

        winsize, err := term.GetWinsize(fd)
        if err == nil {
            termWidth = int(winsize.Width)
            termHeight = int(winsize.Height)
        }
    }

    if err := session.RequestPty("xterm", termHeight, termWidth, modes); err != nil {
        return err
    }

    if len(args) == 0 {
        if err := session.Shell(); err != nil {
            return err
        }

        // monitor for sigwinch
        go monWinCh(session, os.Stdout.Fd())

        session.Wait()
    } else {
        session.Run(strings.Join(args, " "))
    }

    return nil
}

func monWinCh(session *mainssh.Session, fd uintptr) {
    sigs := make(chan os.Signal, 1)

    signal.Notify(sigs, syscall.SIGWINCH)
    defer signal.Stop(sigs)

    // resize the tty if any signals received
    for range sigs {
        session.SendRequest("window-change", false, termSize(fd))
    }
}

func termSize(fd uintptr) []byte {
    size := make([]byte, 16)

    winsize, err := term.GetWinsize(fd)
    if err != nil {
        binary.BigEndian.PutUint32(size, uint32(80))
        binary.BigEndian.PutUint32(size[4:], uint32(24))
        return size
    }

    binary.BigEndian.PutUint32(size, uint32(winsize.Width))
    binary.BigEndian.PutUint32(size[4:], uint32(winsize.Height))

    return size
}