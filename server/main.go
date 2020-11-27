// Package main implements a server for Notification service
package main

import (
	"fmt"
	pb "server-push/notification_proto"
	"log"
	"net"
	"os"
	"io"
	"math/rand"
	"time"
	"strconv"

	"google.golang.org/grpc"
)

const (
	address = "localhost:5001"
)

var ticker *time.Ticker
var clientIDList []ClientIDList

// Server implement the  Notification service
// clients is the map of connected clients
// clientStreams is the map of connected clients stream
type Server struct {
	clients       map[string]*pb.ClientDetail
	clientStreams map[string]*pb.Notification_ConnectToServerServer
	pb.UnimplementedNotificationServer
}

type ClientIDList struct {
	clientID	string      
}

func (server *Server) init() {
	server.clients = make(map[string]*pb.ClientDetail)
	server.clientStreams = make(map[string]*pb.Notification_ConnectToServerServer)
}

// ConnectToServer is called when clietn make connection to the server
// this function will add the client to the servers clienst map and stores the client stream
// the stream should not be killed so we do not return from this server
// for this purpose the infinite loop is used
func (server *Server) ConnectToServer(in *pb.ClientDetail, stream pb.Notification_ConnectToServerServer) error {
	server.addNewClient(in, &stream)
	// loop infinitely to keep stream alive
	// else this stream will be closed
	for {
	}
	return nil
}

// adds new client to map
func (server *Server) addNewClient(in *pb.ClientDetail, stream *pb.Notification_ConnectToServerServer) {
	log.Printf("adding new client")
	server.clientStreams[in.ID] = stream
	server.clients[in.ID] = in
	clientIDList = append(clientIDList, ClientIDList{clientID:in.ID})
	log.Printf("added client with ID: " + in.ID)
	fmt.Println()
}

// send notification to specific client
func (server *Server) sendNotification() {
	
	var (
		writing = true
		buf     []byte
		n       int
		file    *os.File
	)

	noOfClients := len(server.clientStreams)

	file_path := "/home/hisan/Documents/Work/Tasks/POC-GRPC-Go/server-push/server/server-data/swagger.zip"

	randomClient := rand.Intn(noOfClients)
	clientToSend := clientIDList[randomClient].clientID
	log.Println("To Client: " + clientToSend)
	stream := server.clientStreams[clientToSend]

	file, err := os.Open(file_path)

	if err != nil {
		log.Fatalf("failed to open file: %s", file_path)
		return
	}
	defer file.Close()

	buf = make([]byte, 1 << 20)
	var sentBytes = 0
	for writing {
		n, err = file.Read(buf)
		if err != nil {
			if err == io.EOF {
				log.Println("Sent " + strconv.Itoa(sentBytes) + " bytes")
				writing = false
				err = nil
				continue
			}
			log.Fatalf("errored while copying from file to buf")
			return
		}

		err = (*stream).Send(&pb.Chunk{
			Content: buf[:n],
		})
		sentBytes = n
		if err != nil {
			log.Fatalf("failed to send chunk via stream")
			return
		}
	}
}

func main() {

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := &Server{}
	server.init()
	options := []grpc.ServerOption{}
	options = append(options, grpc.MaxMsgSize(100*1024*1024))
	options = append(options, grpc.MaxRecvMsgSize(100*1024*1024))
	s := grpc.NewServer(options...)

	pb.RegisterNotificationServer(s, server)

	// go routine to get server notification message from stdin
	go backgroundTask(server)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

func backgroundTask(server *Server) {
	ticker := time.NewTicker(10 * time.Second)
	rand.Seed(time.Now().UnixNano())
	for _ = range ticker.C {
		log.Println("Timer Triggered")
		noOfClients := len(server.clientStreams)
		log.Println("No of Clients: " + strconv.Itoa(int(noOfClients)))
		if noOfClients > 0 {
			log.Println("Sending Data")
			server.sendNotification()
		} else {
			log.Println("No Clients")
		}
		fmt.Println()
	}
}
