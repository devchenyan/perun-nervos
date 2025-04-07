package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"google.golang.org/grpc"
	"perun.network/channel-service/rpc/proto"
	"perun.network/channel-service/service"
	"perun.network/perun-ckb-backend/backend"
	"perun.network/perun-nervos-demo/deployment"
	"polycry.pt/poly-go/sortedkv/leveldb"
)

const (
	rpcNodeURL = "https://testnet.ckbapp.dev/"
	// rpcNodeURL = "http://localhost:8114"
	network  = types.NetworkTest
	host_A   = ":4322"
	host_B   = ":4323"
	WSSURL_A = "localhost:50051"
	WSSURL_B = "localhost:50052"
)

// 在包级别定义共享的 resolver
// var sharedResolver *service.RelayServerResolver

// func init() {
// 	acc := p2p.NewRandomAccount(rand.New(rand.NewSource(time.Now().UnixNano())))
// 	sharedResolver = service.NewRelayServerResolver(acc)
// }

// SetLogFile sets the log file for the channel service.
func SetLogFile(path string) {
	logFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	log.SetOutput(logFile)
}

func parseSUDTOwnerLockArg(pathToSUDTOwnerLockArg string) (string, error) {
	b, err := os.ReadFile(pathToSUDTOwnerLockArg)
	if err != nil {
		return "", fmt.Errorf("reading sudt owner lock arg from file: %w", err)
	}
	sudtOwnerLockArg := string(b)
	if sudtOwnerLockArg == "" {
		return "", errors.New("sudt owner lock arg not found in file")
	}
	return sudtOwnerLockArg, nil
}

// MakeDeployment creates a deployment object.
func MakeDeployment() (backend.Deployment, error) {
	sudtOwnerLockArg, err := parseSUDTOwnerLockArg("../testnet/accounts/sudt-owner-lock-hash.txt")
	if err != nil {
		fmt.Printf("error getting SUDT owner lock arg: %v", err)
	}
	d, _, err := deployment.GetDeployment("../testnet/contracts/migrations/dev/", "../testnet/system_scripts", sudtOwnerLockArg)
	return d, err
}

func Start(host string, WSSURL string) {
	fmt.Println("Start-------")

	d, err := MakeDeployment()
	if err != nil {
		fmt.Printf("error getting deployment: %v", err)
	}
	// Setup
	db, err := leveldb.LoadDatabase(fmt.Sprintf("./db_%s", host))
	if err != nil {
		fmt.Printf("loading database: %v", err)
	}

	listener, err := net.Listen("tcp", host)
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
		return
	}

	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()

			if err != nil {
				fmt.Printf("accept connection error: %v", err)
				continue
			}

			fmt.Printf("new connection: %s, %s", conn.LocalAddr().String(), conn.RemoteAddr().String())
		}
	}()

	// neuron里创建的
	walletServiceClient := setupWalletServiceClient(WSSURL)

	channelService, err := service.NewChannelService(walletServiceClient, network, rpcNodeURL, d, nil, db)
	if err != nil {
		fmt.Printf("creating channel service: %v", err)
	}

	// 在这里 创建 ChannelServiceServer 服务器
	var opts []grpc.ServerOption

	channelServiceServer := grpc.NewServer(opts...)
	proto.RegisterChannelServiceServer(channelServiceServer, channelService)

	// 启动服务
	fmt.Printf("start channel service gRPC server, listening on: %s\n", host)

	err = channelServiceServer.Serve(listener)

	if err != nil {
		fmt.Printf("serving channel service: %v", err)
	}

}

// Start channel service GRPC server.
func main() {
	SetLogFile("channel_service.log")

	go Start(host_B, WSSURL_B)

	select {}
}
