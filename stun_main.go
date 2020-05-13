package main

import (
    "fmt"
    "github.com/pion/stun"
)


func main() {
    stun_serv, err := stun.Dial("udp", "stun.l.google.com:19302")
    if err != nil {
        fmt.Println("Stun Dial error")
    }

    msg := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

    // send msg to stun server, wait for message request 
    stun_serv.Do(msg, func(res stun.Event) {
        //if res.Error != nil {
        //    return res.Error
        //}
        var xorAddr stun.XORMappedAddress
        if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
            //return getErr;
            fmt.Println("Error in GetFrom!")
        }
        fmt.Println(xorAddr)
    })
    stun_serv.Close()


}



