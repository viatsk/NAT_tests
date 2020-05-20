package main

import (
    "fmt"
    "net"
    "github.com/pion/stun"
)


func main() {
//    udp_serv := stun.NewUDPServer(nil, "test", 255, <handler>)
    stun_serv, err := stun.Dial("udp", "stun.l.google.com:19302")
    if err != nil {
        fmt.Println("Stun Dial error")
    }

    m := new(stun.Message)
    addr := &stun.XORMappedAddress {
        IP: net.IPv4(1,2,3,4),
    }
    addr.AddTo(m)
    msg := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
    fmt.Println(msg)
    fmt.Println(m)

    // send msg to stun server, wait for message request 
    stun_serv.Do(msg, func(res stun.Event) {
        if res.Error != nil {
            fmt.Println("Error in res", res.Error)
        }
        fmt.Println(res.TransactionID)
        fmt.Println(res.Message)
        var xorAddr stun.XORMappedAddress
        if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
            //return getErr;
            fmt.Println("Error in GetFrom!")
        }
        fmt.Println("xorAddr is", xorAddr)
    })
    stun_serv.Close()


}



