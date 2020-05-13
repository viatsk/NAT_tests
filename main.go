package tmp

import (
    "sync/atomic"
    "fmt"
    "time"
    "net"
    "github.com/pion/logging"
    "github.com/pion/transport/vnet"
    "github.com/pion/webrtc/v2"
//    "github.com/pion/stun"
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

/*
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
    return nil
}
*/

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
    // Make new router that is root router
    rootRouter, err := makeRouter("") // Type doesnt matter.... 
    secondRouter, err := makeRouter(S1)

    // Add chunk filter (monitors the traffic on the router)
    // on root or second router? 
    rootRouter.AddChunkFilter(chunkFilter)

    // LOG THROUGHPUT?? 

    // create a network interface 
    // can specify static IPs for the instance of the Net to use
    // if not specified, router will assign an IP address that is contained in the router's CIDR
    // this network interface is for the OFFERER 
    network := vnet.NewNet(&vnet.NetConfig {
        StaticIPs: []string{"1.2.3.4"},
    })


    // add the network to the router; the router will assign new IPs to network
    // this calls addNIC internally 
    errHandler(rootRouter.AddNet(network))
    errHandler(secondRouter.AddNet(network))

    offerSettingEngine := webrtc.SettingEngine{}
    offerSettingEngine.SetVNet(network)
    offerAPI := webrtc.NewAPI(webrtc.WithSettingEngine(offerSettingEngine))

    // create an answer VNET 
    answerVNet := vnet.NewNet(&vnet.NetConfig {
        StaticIPs: []string{"1.2.3.5"}, 
    })
    errHandler(rootRouter.AddNet(network))
    errHandler(secondRouter.AddNet(network))

    answerSettingEngine := webrtc.SettingEngine{}
    answerSettingEngine.SetVNet(answerVNet)
    answerAPI := webrtc.NewAPI(webrtc.WithSettingEngine(answerSettingEngine))


   // NOW WE SEND MESSAGES WEEE 
    offerPeerConnection, err := offerAPI.NewPeerConnection(webrtc.Configuration{})
    answerPeerConnection, err := answerAPI.NewPeerConnection(webrtc.Configuration{})
    offerDataChannel, err := offerPeerConnection.CreateDataChannel("channel", nil)


/* OLD 
    // Create a Network Interface Card
    nic := make([]*dummyNIC, 2)
    ip  := make([]*net.UDPAddr, 2)

    // need to interact with the NIC interfaces  
    for i := 0; i < 2; i++ {
        anic := vnet.NewNet(&vnet.NetConfig{})
        nic[i] = &dummyNIC {
            Net: *anic,
        }

        // Add dummy NIC card to the second router only 
        err2 := secondRouter.AddNet(nic[i])

        eth0, err2  := nic[i].InterfaceByName("eth0")
        addrs, err2 := eth0.Addrs()
        ip[i] = &net.UDPAddr {
            IP: addrs[0].(*net.IPNet).IP,
            Port: 1111 * (i+1),
        }

        nic[i].onInboundChunkHandler = func (c vnet.Chunk) {
            p("nic[%d] received: %s", i, c.String())
            p(c.UserData()[0])
        }
        p(err2)
    }
*/

    // Start router, will start internal goroutine to route packets 
    // call on root router, will propogate to children  
    err = rootRouter.Start();

    // SEND 3 PACKETS
//    for i := 0; i < 3; i++ {
//        c := newChunkUDP(ip[0], ip[1])
//        c.userData = make([]byte, 1)
//        c.userData[0] = byte(i)
    //    secondRouter.push(c)
        // Cant do push --- what to do? 
//    }

    time.Sleep(30*time.Millisecond)

    secondRouter.Stop()
    p(rootRouter, err)
    p(secondRouter, err)
    p("Done")

}

