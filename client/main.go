// Package main implements a client for Notification service
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	pb "server-push/notification_proto"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"gopkg.in/yaml.v2"
)

const (
	address = "localhost:443"
)

// YamlConfig Config to extract Yaml Content
type YamlConfig struct {
	Info struct {
		Title string `yaml:"title"`
	} `yaml:"info"`
}

func main() {

	// Uncomment if using environment variable approach to get total clients
	// obtain total clients as an environment variable
	// totalClients, err := strconv.Atoi(os.Getenv("TOTAL_CLIENTS"))
	// if err != nil {
	// 	log.Print(err)
	// 	log.Fatalf("Enter valid number for TOTAL_CLIENTS %v", err)
	// }

	// obtain total clients as a flag passed from command line
	clientsPtr := flag.Int("clients", 0, "Total Number of Clients")
	flag.Parse()
	totalClients := *clientsPtr

	// create specified number of clients
	for i := 0; i < totalClients; i++ {
		time.Sleep(1 * time.Second)
		uuidWithHyphen := uuid.New()
		uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)

		clientDetails := &pb.ClientDetail{
			ID: uuid,
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
	//conn, err := grpc.Dial(address, grpc.WithInsecure())
	//ctx, cancel := context.WithTimeout(context.Background(), 100000*time.Millisecond)
	//defer cancel()
	h2creds := credentials.NewTLS(&tls.Config{NextProtos: []string{"h2"}, InsecureSkipVerify: true})
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(h2creds), grpc.WithKeepaliveParams(keepalive.ClientParameters{Time: 10 * time.Second, Timeout: 86400 * time.Second}))
	if err != nil {
		log.Fatalf("did not dial: %v", err)
	}
	client := pb.NewNotificationClient(conn)

	stream, err := client.ConnectToServer(context.Background(), clientDetails)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	// background task to receive messages from the server
	go receiveMessage(clientDetails, stream)
}

func receiveMessage(clientDetails *pb.ClientDetail, stream pb.Notification_ConnectToServerClient) {
	content := bytes.Buffer{}
	receivedSize := 0

	for {
		// listen for streams
		message, err := stream.Recv()
		if err != nil {
			log.Fatalf("Error in receiving message for client: "+clientDetails.ID+" - %v", err)
		}

		if message.Status == pb.TransferStatusCode_Completed {
			log.Println("Receive Complete")
			log.Println("Total Received bytes: " + strconv.Itoa(receivedSize))

			// open a zip archive for reading.
			reader := bytes.NewReader(content.Bytes())
			zipReader, err := zip.NewReader(reader, int64(receivedSize))
			if err != nil {
				log.Fatal(err)
			}

			for _, f := range zipReader.File {
				log.Println("Reading Contents of " + f.Name)

				// reading the content of the file
				contentReader, err := f.Open()
				if err != nil {
					log.Fatal(err)
				}
				buffer := new(bytes.Buffer)
				buffer.ReadFrom(contentReader)
				data := buffer.Bytes()

				// extracting the title from the content
				var yamlConfig YamlConfig
				err = yaml.Unmarshal(data, &yamlConfig)
				if err != nil {
					log.Printf("Error parsing YAML file: %v\n", err)
				}
				log.Println("Title: " + yamlConfig.Info.Title)
				fmt.Println()
				content = bytes.Buffer{}
				receivedSize = 0
				contentReader.Close()
			}
		} else if message.Status == pb.TransferStatusCode_Pending {
			if receivedSize == 0 {
				log.Println("Receiving client ID: " + clientDetails.ID)
			}
			receivedSize += len(message.Content)
			_, err = content.Write(message.Content)
			if err != nil {
				log.Printf("Error in writing content: %v\n", err)
			}
			log.Println("Received " + strconv.Itoa(len(message.Content)) + " bytes")
		}

		if err == io.EOF { //no more stream to listen
			break
		}
		if err != nil { // some error occured
			log.Fatalf("%v", err)
		}
	}
}
