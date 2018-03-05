package main


import (
	"bytes"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol/codec"
	"gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"


	"gitlab.alipay-inc.com/afe/mosn/pkg/network"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server/config/proxy"
	"gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
	"time"
	"net"
)

const (
	RealRPCServerAddr = "127.0.0.1:8088"
	MeshRPCServerAddr = "127.0.0.1:2044"
	TestClusterRPC    = "tstCluster"
	TestListenerRPC   = "tstListener"
)


func main() {

	//test_codec()
	//initilize codec engine. TODO: config driven
	//codecImpl := codec.NewProtocols(map[byte]protocol.Protocol{
	//	sofarpc.PROTOCOL_CODE_V1:sofarpc.BoltV1,
	//	sofarpc.PROTOCOL_CODE_V2:sofarpc.BoltV2,
	//	sofarpc.PROTOCOL_CODE:sofarpc.Tr,
	//
	//})

	stopChan := make(chan bool)
	upstreamReadyChan := make(chan bool)
	meshReadyChan := make(chan bool)


	go func() {
		// upstream
		l, _ := net.Listen("tcp", RealRPCServerAddr)

		defer l.Close()

		for {
			select {
			case <-stopChan:
				break
			default:
				upstreamReadyChan <- true

				conn, _ := l.Accept()

				fmt.Printf("[REALSERVER]get connection %s..", conn.RemoteAddr())
				fmt.Println()

				buf := make([]byte, 4*1024)

				for {
					t := time.Now()
					conn.SetReadDeadline(t.Add(3 * time.Second))

					if bytesRead, err := conn.Read(buf); err != nil {

						if err, ok := err.(net.Error); ok && err.Timeout() {
							continue
						}

						fmt.Println("[REALSERVER]failed read buf")
						return
					} else {
						if bytesRead > 0 {
							fmt.Printf("[REALSERVER]get data '%s'", string(buf[:bytesRead]))
							fmt.Println()
							break
						}
					}
				}

				fmt.Printf("[REALSERVER]write back data 'Got Bolt Msg'")
				fmt.Println()

				conn.Write([]byte("Got Bolt"))

				select {
				case <-stopChan:
					conn.Close()
				}
			}
		}
	}()

	go func() {
		select {
		case <-upstreamReadyChan:
			//  mesh
			cmf := &clusterManagerFilterRPC{}

			//RPC
			srv := server.NewServer(&proxy.RpcProxyFilterConfigFactory{
				Proxy: rpcProxyConfig(),
			}, cmf)

	//		boltV1PostData := bytes.NewBuffer([]byte("\x01\x00BoltV1"))
			//codecImpl.Decode(nil,boltV1PostData,nil)

			//

			srv.AddListener(rpcProxyListener())
			cmf.cccb.UpdateClusterConfig(clustersrpc())
			cmf.chcb.UpdateClusterHost(TestClusterRPC, 0, rpchosts())

			meshReadyChan <- true

			srv.Start()

			select {
			case <-stopChan:
				srv.Close()
			}
		}
	}()

	go func() {
		select {
		case <-meshReadyChan:
			// client
			remoteAddr, _ := net.ResolveTCPAddr("tcp", MeshRPCServerAddr)
			cc := network.NewClientConnection(nil, remoteAddr, stopChan)
			cc.AddConnectionCallbacks(&rpclientConnCallbacks{      //ADD  connection callback
				cc: cc,
			})
			cc.Connect()
			cc.SetReadDisable(false)
			cc.FilterManager().AddReadFilter(&rpcclientConnReadFilter{})

			select {
			case <-stopChan:
				cc.Close(types.NoFlush,types.LocalClose)
			}
		}
	}()

	select {
	case <-time.After(time.Second * 5):
		stopChan <- true
		fmt.Println("[MAIN]closing..")
	}





}
func rpcProxyConfig() *v2.RpcProxy {
	rpcProxyConfig := &v2.RpcProxy{}
	rpcProxyConfig.Routes = append(rpcProxyConfig.Routes, &v2.RpcRoute{
		Cluster: TestClusterRPC,
	})

	return rpcProxyConfig
}

func rpcProxyListener() v2.ListenerConfig {
	return v2.ListenerConfig{
		Name:                 TestListenerRPC,
		Addr:                 MeshRPCServerAddr,
		BindToPort:           true,
		ConnBufferLimitBytes: 1024 * 32,
	}
}

func rpchosts() []v2.Host {
	var hosts []v2.Host

	hosts = append(hosts, v2.Host{
		Address: RealRPCServerAddr,
		Weight:  100,
	})

	return hosts
}


//
func test_codec(){

	//initilize codec engine. TODO: config driven
	codecImpl := codec.NewProtocols(map[byte]protocol.Protocol{
		protocol.PROTOCOL_CODE_V1:sofarpc.BoltV1,
		protocol.PROTOCOL_CODE_V2:sofarpc.BoltV2,
		//protocol.PROTOCOL_CODE:sofarpc.Tr,

	})

	fakePipe := make(chan protocol.RpcCommand)

	//plug-in tr codec
	codecImpl.PutProtocol(protocol.PROTOCOL_CODE, sofarpc.Tr)

	boltV1EchoBytes := []byte{0x01, 0x01, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x72, 0x01, 0x00, 0x00, 0x00, 0x64, 0x00, 0x2c, 0x00, 0x45, 0x00, 0x00, 0x01, 0xe0, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x6c, 0x69, 0x70, 0x61, 0x79, 0x2e, 0x73, 0x6f, 0x66, 0x61, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x6f, 0x66, 0x61, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x00, 0x00, 0x00, 0x07, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x00, 0x00, 0x00, 0x36, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x6c, 0x69, 0x70, 0x61, 0x79, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x66, 0x61, 0x63, 0x61, 0x64, 0x65, 0x2e, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x3a, 0x31, 0x2e, 0x30, 0x4f, 0xbc, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x6c, 0x69, 0x70, 0x61, 0x79, 0x2e, 0x73, 0x6f, 0x66, 0x61, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x63, 0x6f, 0x72, 0x65, 0x2e, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x6f, 0x66, 0x61, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x95, 0x0d, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x41, 0x70, 0x70, 0x4e, 0x61, 0x6d, 0x65, 0x0a, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x4e, 0x61, 0x6d, 0x65, 0x17, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x55, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x0c, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x50, 0x72, 0x6f, 0x70, 0x73, 0x0d, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x41, 0x72, 0x67, 0x53, 0x69, 0x67, 0x73, 0x6f, 0x90, 0x07, 0x72, 0x70, 0x63, 0x2d, 0x62, 0x61, 0x72, 0x07, 0x65, 0x63, 0x68, 0x6f, 0x53, 0x74, 0x72, 0x53, 0x00, 0x36, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x6c, 0x69, 0x70, 0x61, 0x79, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x66, 0x61, 0x63, 0x61, 0x64, 0x65, 0x2e, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x3a, 0x31, 0x2e, 0x30, 0x4d, 0x11, 0x72, 0x70, 0x63, 0x5f, 0x74, 0x72, 0x61, 0x63, 0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x4d, 0x09, 0x73, 0x6f, 0x66, 0x61, 0x52, 0x70, 0x63, 0x49, 0x64, 0x01, 0x30, 0x07, 0x45, 0x6c, 0x61, 0x73, 0x74, 0x69, 0x63, 0x01, 0x46, 0x0b, 0x73, 0x79, 0x73, 0x50, 0x65, 0x6e, 0x41, 0x74, 0x74, 0x72, 0x73, 0x00, 0x0d, 0x73, 0x6f, 0x66, 0x61, 0x43, 0x61, 0x6c, 0x6c, 0x65, 0x72, 0x49, 0x64, 0x63, 0x03, 0x64, 0x65, 0x76, 0x09, 0x7a, 0x70, 0x72, 0x6f, 0x78, 0x79, 0x55, 0x49, 0x44, 0x00, 0x10, 0x7a, 0x70, 0x72, 0x6f, 0x78, 0x79, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x5a, 0x6f, 0x6e, 0x65, 0x00, 0x0c, 0x73, 0x6f, 0x66, 0x61, 0x43, 0x61, 0x6c, 0x6c, 0x65, 0x72, 0x49, 0x70, 0x0d, 0x31, 0x31, 0x2e, 0x31, 0x36, 0x36, 0x2e, 0x32, 0x32, 0x2e, 0x31, 0x36, 0x31, 0x0b, 0x73, 0x6f, 0x66, 0x61, 0x54, 0x72, 0x61, 0x63, 0x65, 0x49, 0x64, 0x1e, 0x30, 0x62, 0x61, 0x36, 0x31, 0x36, 0x61, 0x31, 0x31, 0x35, 0x31, 0x34, 0x34, 0x33, 0x35, 0x33, 0x37, 0x31, 0x39, 0x36, 0x32, 0x31, 0x30, 0x30, 0x34, 0x34, 0x38, 0x30, 0x30, 0x35, 0x0c, 0x73, 0x6f, 0x66, 0x61, 0x50, 0x65, 0x6e, 0x41, 0x74, 0x74, 0x72, 0x73, 0x00, 0x0e, 0x73, 0x6f, 0x66, 0x61, 0x43, 0x61, 0x6c, 0x6c, 0x65, 0x72, 0x5a, 0x6f, 0x6e, 0x65, 0x05, 0x47, 0x5a, 0x30, 0x30, 0x42, 0x0d, 0x73, 0x6f, 0x66, 0x61, 0x43, 0x61, 0x6c, 0x6c, 0x65, 0x72, 0x41, 0x70, 0x70, 0x07, 0x72, 0x70, 0x63, 0x2d, 0x66, 0x6f, 0x6f, 0x0d, 0x7a, 0x70, 0x72, 0x6f, 0x78, 0x79, 0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x03, 0x31, 0x30, 0x30, 0x7a, 0x7a, 0x56, 0x74, 0x00, 0x07, 0x5b, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x6e, 0x01, 0x10, 0x6a, 0x61, 0x76, 0x61, 0x2e, 0x6c, 0x61, 0x6e, 0x67, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x7a, 0x01, 0x61}


	trPostData := bytes.NewBuffer([]byte("\x0d\x00TaobaoRemoting"))
	boltV1PostData := bytes.NewBuffer(boltV1EchoBytes)
	boltV2PostData := bytes.NewBuffer([]byte("\x02\x00BoltV2"))

	//test tr decode branch
	fmt.Println("-----------> tr test begin")
	codecImpl.Decode(nil, trPostData, nil)
	fmt.Println("<----------- tr test end\n")

	//test boltv1 decode branch
	fmt.Println("-----------> boltv1 test begin")
	go codecImpl.Decode(nil, boltV1PostData, fakePipe)
	fmt.Println("<----------- boltv1 test end\n")

	//test boltv2 decode branch
	fmt.Println("-----------> boltv2 test begin")
	codecImpl.Decode(nil, boltV2PostData, nil)
	fmt.Println("<----------- boltv2 test end\n")

	msg := <-fakePipe
	codecImpl.Handle(1, nil, msg)
}

type clusterManagerFilterRPC struct {
	cccb server.ClusterConfigFactoryCb
	chcb server.ClusterHostFactoryCb
}


func (cmf *clusterManagerFilterRPC) OnCreated(cccb server.ClusterConfigFactoryCb, chcb server.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}


func clustersrpc() []v2.Cluster {
	var configs []v2.Cluster
	configs = append(configs, v2.Cluster{
		Name:                 TestClusterRPC,
		ClusterType:          v2.SIMPLE_CLUSTER,
		LbType:               v2.LB_RANDOM,
		MaxRequestPerConn:    1024,
		ConnBufferLimitBytes: 16 * 1026,
	})

	return configs
}


type rpclientConnCallbacks struct {
	cc types.Connection
}

func (ccc *rpclientConnCallbacks) OnEvent(event types.ConnectionEvent) {
	fmt.Printf("[CLIENT]connection event %s", string(event))
	fmt.Println()

	switch event {
	case types.Connected:
		time.Sleep(3 * time.Second)

		fmt.Println("[CLIENT]write 'bolt test msg' to remote server")

		//buf := bytes.NewBufferString("hello")
	//	boltV1PostData := bytes.NewBuffer([]byte("\x01\x00BoltV1test"))




		//t:=types.IoBuffer(boltV1PostData.Bytes())
		//ccc.cc.Write(buf)
		boltV1PostData := buffer.NewIoBufferString("\x01\x00BoltV1test")
		//boltV1PostData := &buffer.IoBuffer{: []byte([]byte("\x01\x00BoltV1test"))}
		ccc.cc.Write(boltV1PostData)

	}
}

func (ccc *rpclientConnCallbacks) OnAboveWriteBufferHighWatermark() {}

func (ccc *rpclientConnCallbacks) OnBelowWriteBufferLowWatermark() {}


type rpcclientConnReadFilter struct {
}

func (ccrf *rpcclientConnReadFilter) OnData(buffer types.IoBuffer) types.FilterStatus {
	fmt.Printf("[CLIENT]receive data '%s'", buffer.String())
	fmt.Println()
	buffer.Reset()

	return types.Continue
}

func (ccrf *rpcclientConnReadFilter) OnNewConnection() types.FilterStatus {
	return types.Continue
}

func (ccrf *rpcclientConnReadFilter) InitializeReadFilterCallbacks(cb types.ReadFilterCallbacks) {}