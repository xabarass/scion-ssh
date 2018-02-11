package scionutils

import (
    "fmt"

    "github.com/scionproto/scion/go/lib/snet"
)

func GetSciondAddr(scionAddr *snet.Addr)(string){
    return fmt.Sprintf("/run/shm/sciond/sd%d-%d.sock", scionAddr.IA.I, scionAddr.IA.A)
}

func GetDispatcherAddr(scionAddr *snet.Addr)(string){
    return "/run/shm/dispatcher/default.sock"
}

