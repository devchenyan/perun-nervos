package main

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"perun.network/channel-service/rpc/proto"
)

func setupWalletServiceClient(url string) proto.WalletServiceClient {
	conn, err := grpc.Dial(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("\nfailed to dial: %v", err)
	}

	client := proto.NewWalletServiceClient(conn)

	// Create a goroutine to monitor the connection and redial if necessary
	go func() {
		for {
			if conn.GetState() == connectivity.TransientFailure {
				fmt.Println("WalletServiceClient: Connection lost. Reconnecting...")
				for {
					if conn.GetState() != connectivity.TransientFailure {
						fmt.Println("WalletServiceClient: Reconnection successful!")
						break
					}
					time.Sleep(1 * time.Second) // Adjust the retry interval as needed
					conn, err = grpc.Dial(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
					if err != nil {
						fmt.Printf("WalletServiceClient:Error reconnecting: %v\n", err)
					} else {
						client = proto.NewWalletServiceClient(conn)
					}
				}
			}
			// fmt.Println("WalletServiceClient: Connection state: ", conn.GetState())
			time.Sleep(1 * time.Second) // Check connection state every second
		}
	}()

	return client
}
