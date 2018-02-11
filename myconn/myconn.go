package myconn

import (
    "time"
    "net"
    quic "github.com/lucas-clemente/quic-go"
)

type MyConn struct {
    Session quic.Session
    Stream quic.Stream
}

func (mc *MyConn)Read(b []byte) (n int, err error){
    return mc.Stream.Read(b)
}

func (mc *MyConn) Write(b []byte) (n int, err error){
    return mc.Stream.Write(b)   
}

func (mc *MyConn) Close() error {
    return mc.Stream.Close()
}

func (mc *MyConn) LocalAddr() net.Addr{
    return mc.Session.LocalAddr()
}

func (mc *MyConn) RemoteAddr() net.Addr{
    return mc.Session.RemoteAddr()
}

func (mc *MyConn) SetDeadline(t time.Time) error{
    return mc.Stream.SetDeadline(t)
}

func (mc *MyConn) SetReadDeadline(t time.Time) error{
    return mc.Stream.SetReadDeadline(t)
}

func (mc *MyConn) SetWriteDeadline(t time.Time) error{
    return mc.Stream.SetWriteDeadline(t)
}