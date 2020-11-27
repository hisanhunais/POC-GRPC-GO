// Package main implements a client for Notification service
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	pb "server-push/notification_proto"
	"google.golang.org/grpc"
	"github.com/google/uuid"
	"strconv"
	"time"
	"strings"
	//"os"
	"flag"
)

const (
	address = "localhost:5001"
)

func main() {

	// totalClients, err := strconv.Atoi(os.Getenv("TOTAL_CLIENTS"))
	// if err != nil {
	// 	log.Print(err)
	// 	log.Fatalf("Enter valid number for TOTAL_CLIENTS %v", err)
	// }

	clientsPtr := flag.Int("clients", 0, "Total Number of Clients")
	flag.Parse()
	totalClients := *clientsPtr

	for i := 0; i < totalClients; i++ {
		time.Sleep(6 * time.Second)
		uuidWithHyphen := uuid.New()
		uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)
		
		clientDetails := &pb.ClientDetail{
			ID:    uuid,
		}
		log.Println("Client with ID: " + clientDetails.ID + " created")
		fmt.Println()
		connectToServer(clientDetails)
	}
	log.Println("Finished creating " + strconv.Itoa(totalClients) + " clients")
	fmt.Println()
	for {
	}

	
}

func connectToServer(clientDetails *pb.ClientDetail) {

	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	client := pb.NewNotificationClient(conn)

	stream, err := client.ConnectToServer(context.Background(), clientDetails)
	go receiveMessage(clientDetails, stream)
	
}

func receiveMessage(clientDetails *pb.ClientDetail, stream pb.Notification_ConnectToServerClient) {
	for {
		// listen for streams
		message, err := stream.Recv()
		log.Println("Receiving client ID: " + clientDetails.ID)
		log.Println("Received " + strconv.Itoa(len(message.Content)) + " bytes")

		if err == io.EOF { //no more stream to listen
			break
		}
		if err != nil { // some error occured
			log.Fatalf("%v", err)
		}
		log.Println("Receive Complete")
		fmt.Println("") 
	}
}
