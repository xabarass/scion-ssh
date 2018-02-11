package ssh

import(
    "log"
        "os/exec"
        "github.com/kr/pty"
        "sync"
        "io"
        "syscall"
        "encoding/binary"
        "unsafe"
    
    "golang.org/x/crypto/ssh"
)

func handleSession(newChannel ssh.NewChannel){
    connection, requests, err := newChannel.Accept()
    if err != nil {
        log.Printf("Could not accept channel (%s)", err)
        return
    }

    bash := exec.Command("bash")

    // Prepare teardown function
    close := func() {
        connection.Close()
        _, err := bash.Process.Wait()
        if err != nil {
            log.Printf("Failed to exit bash (%s)", err)
        }
        log.Printf("Session closed")
    }

    // Allocate a terminal for this channel
    log.Print("Creating pty...")
    bashf, err := pty.Start(bash)
    if err != nil {
        log.Printf("Could not start pty (%s)", err)
        close()
        return
    }

    //pipe session to bash and visa-versa
    var once sync.Once
    go func() {
        io.Copy(connection, bashf)
        once.Do(close)
    }()
    go func() {
        io.Copy(bashf, connection)
        once.Do(close)
    }()

    // Sessions have out-of-band requests such as "shell", "pty-req" and "env"
    go func() {
        for req := range requests {
            switch req.Type {
            case "shell":
                // We only accept the default shell
                // (i.e. no command in the Payload)
                if len(req.Payload) == 0 {
                    req.Reply(true, nil)
                }
            case "pty-req":
                termLen := req.Payload[3]
                w, h := parseDims(req.Payload[termLen+4:])
                SetWinsize(bashf.Fd(), w, h)
                // Responding true (OK) here will let the client
                // know we have a pty ready for input
                req.Reply(true, nil)
            case "window-change":
                w, h := parseDims(req.Payload)
                SetWinsize(bashf.Fd(), w, h)
            }
        }
    }()
}

// parseDims extracts terminal dimensions (width x height) from the provided buffer.
func parseDims(b []byte) (uint32, uint32) {
    w := binary.BigEndian.Uint32(b)
    h := binary.BigEndian.Uint32(b[4:])
    return w, h
}

// ======================

// Winsize stores the Height and Width of a terminal.
type Winsize struct {
    Height uint16
    Width  uint16
    x      uint16 // unused
    y      uint16 // unused
}

// SetWinsize sets the size of the given pty.
func SetWinsize(fd uintptr, w, h uint32) {
    ws := &Winsize{Width: uint16(w), Height: uint16(h)}
    syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
}