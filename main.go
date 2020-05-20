package main

import (
    "sync/atomic"
    "fmt"
    "time"
    "net"
    "github.com/pion/logging"
    "github.com/pion/transport/vnet"
    //"github.com/pion/webrtc/v2"
//    "github.com/pion/stun"
    "flag"
)

var p = fmt.Println
var FC  = "fullCone"
var RC  = "restCone"
var PRC = "portRestCone"
var S1   = "symmetric1"
var S2   = "symmetric2"

var inboundBytes int32
var outboundBytes int32
/*
 +-----------------------------------------+
 |                VNet                     |
 |                                         |
 |                                         |
 +-----------------------------------------+



*/

func errHandler(err error) {
    if err != nil {
        panic(err)
    }
}


type dummyNIC struct {
    vnet.Net
    onInboundChunkHandler func(vnet.Chunk)
}

type chunkUDP struct {
    sourcePort      int
    destinationPort int
    userData        []byte
}

type AgentConfig struct {
    Net *vnet.Net
}

type Agent struct {
    net *vnet.Net
}

func NetAgent(config *AgentConfig) *Agent {
    if config.Net == nil {
        config.Net = vnet.NewNet(nil) // defaults to native operation 
    }
    return &Agent {
        net: config.Net,
    }
}

func (a *Agent) listenUDP (address string) error {
    // a.net is an instance of vnet.NET class 
    conn, err := a.net.ListenPacket("udp", address)
    if err != nil {
        return err
    }
    p("connection is ", conn)
    return nil
}

func getNATType (inp string) (*vnet.NATType) {
    var mapBehaviour, filBehaviour vnet.EndpointDependencyType

    switch inp {
        case FC:
            mapBehaviour = vnet.EndpointIndependent
            filBehaviour = vnet.EndpointIndependent
        case RC:
            mapBehaviour = vnet.EndpointIndependent
            filBehaviour = vnet.EndpointAddrDependent
        case PRC:
            mapBehaviour = vnet.EndpointIndependent
            filBehaviour = vnet.EndpointAddrPortDependent
        case S1:
            mapBehaviour = vnet.EndpointAddrDependent
            filBehaviour = vnet.EndpointAddrDependent
        case S2:
            mapBehaviour = vnet.EndpointAddrDependent
            filBehaviour = vnet.EndpointAddrDependent
        default:
            return nil // for root routers
    }

    return &vnet.NATType{
        Mode             : vnet.NATModeNormal,
        MappingBehavior  : mapBehaviour,
        FilteringBehavior: filBehaviour,
        Hairpining       : false, // not yet implemented 
        MappingLifeTime  : 30*time.Second, // ? 
    }

}


func makeRouter (inp string) (*vnet.Router, error) {
    loggerFactory := logging.NewDefaultLoggerFactory()
    fmt.Println(inp)

    newNatType := getNATType(inp)

    newRouterConf := &vnet.RouterConfig{
        Name : inp,
        CIDR : "1.2.3.0/24",
       // StaticIPs : []net.IP{net.ParseIP("1.2.3.4")},
        QueueSize : 30,
        NATType   : newNatType,
        LoggerFactory: loggerFactory,
    }

    return vnet.NewRouter(newRouterConf)
}

/*
func newChunkUDP (srcAddr, dstAddr *net.UDPAddr) *chunkUDP {
    return &chunkUDP {
        //chunkIP: chunkIP {
        //    sourceIP: srcAddr.IP,
        //    destinationIP: dstAddr.IP,
        //},
        sourcePort: srcAddr.Port,
        destinationPort: dstAddr.Port,
    }
}
*/

func chunkFilter (c vnet.Chunk) bool {
    netType := c.SourceAddr().Network() // should be "udp"
    p(netType)
    dstAddr := c.DestinationAddr().String()
    host, _ , err := net.SplitHostPort(dstAddr)
    errHandler(err)

    p("chunk Filter host is :", host)
    if host == "1.2.3.4" {
        // if the host is 1.2.3.4, we need to return a []byte of UDP payload
        atomic.AddInt32(&inboundBytes, int32(len(c.UserData())))
    }
    srcAddr := c.SourceAddr().String()
    host, _, err = net.SplitHostPort(srcAddr)
    if host == "1.2.3.4" {
        // user data returns a byte of the UDP payload ... 
        atomic.AddInt32(&outboundBytes, int32(len(c.UserData())))
    }

    return true
}

// TODO:
// The root router will ignore the configs - make sure there are two non-route routers here, then
// Root router has no NAT - why 
func main() {
    var (
        server = flag.String("server", fmt.Sprintf("pion.ly:3478"), "Stun Server Address")
    )
    p("SERVER IS ", *server)
    maxMessageSize := 512
    // Make new router that is root router
    rootRouter, err := makeRouter("") // Type doesnt matter.... 
    secondRouter, err := makeRouter(S1)

    // Add chunk filter (monitors the traffic on the router)
    // on root or second router? 
    rootRouter.AddChunkFilter(chunkFilter)

    // create a network interface 
    // can specify static IPs for the instance of the Net to use
    // if not specified, router will assign an IP address that is contained in the router's CIDR
    // this network interface is for the OFFERER 
    offerVNet := vnet.NewNet(&vnet.NetConfig {
        StaticIPs: []string{"1.2.3.4"},
    })

    // TODO: Use the bridge type outlined in the transport class
    // add the network to the router; the router will assign new IPs to network; this calls addNIC internally 
    errHandler(rootRouter.AddNet(offerVNet))
    errHandler(secondRouter.AddNet(offerVNet))

    doneCh := make(chan struct{})

    // Start router, will start internal goroutine to route packets 
    // call on root router, will propogate to children  
    err = rootRouter.Start();

    srvAddr, err := net.ResolveUDPAddr("udp", *server)
    c, err := offerVNet.ListenPacket("udp", "1.2.3.4:1234") // may prefer ListenPacket? 

    p("server address is ", srvAddr)
    p("connection returned by listenUDP is", c)

    conn1RcvdCh := make(chan bool)
    go func() {
        buf := make ([]byte, maxMessageSize)
        for {
            p("waitin for a message.... ")
            n, _, err2 := c.ReadFrom(buf)
            if (err2 != nil) {
                p("ERROR: ReadFrom returned : %v", err2)
                break
            }
            p("offer recieved %s", string(buf[:n]))
            conn1RcvdCh <- true
        }
        close(doneCh)
    }()

    answerVNet := vnet.NewNet(&vnet.NetConfig {
        StaticIPs: []string{"1.2.3.5"},
    })
    errHandler(rootRouter.AddNet(answerVNet))
    errHandler(secondRouter.AddNet(answerVNet))
    c2, err := answerVNet.ListenPacket("udp", "1.2.3.5:1234")

    go func() {
        buf := make([]byte, maxMessageSize)
        for {
            p("c2 waiting for message... ")
            n, addr, err2 := c2.ReadFrom(buf)
            if err2 != nil {
                p("ERROR: ReadFrom2 returned %v", err2)
                break
            }
            p("answer recieved %s", string(buf[:n]))

            // Send something back to c
            nSent, err2 := c2.WriteTo([]byte("Goodbye"), addr)
            p(nSent, err2)
        }
    }()

    p("sending to c!")
    nSent, err := c.WriteTo(
        []byte("Hello"),
        c2.LocalAddr(),
    )
    p("nSent is ", nSent)

 loop:
    for {
        select {
        case <-conn1RcvdCh:
            c.Close()
            c2.Close()
        case <-doneCh:
            break loop
        }
    }

/*
    go func() {
        duration := 3*time.Second
        for {
            time.Sleep(duration)
            inBytes := atomic.SwapInt32(&inboundBytes, 0)
            outBytes := atomic.SwapInt32(&outboundBytes, 0)
            p("inbound tp ", float64(inBytes) / duration.Seconds())
            p("outbound tp ", float64(outBytes) / duration.Seconds())
        }
    } ()


    time.Sleep(30*time.Millisecond)

    secondRouter.Stop()
*/
    p(rootRouter, err)
    p(secondRouter, err)
    p("Done")

}

