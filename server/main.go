package main

import (
	"log"
	"net"
	"os"
	"io"

	"google.golang.org/grpc"
	pb "fl-server/server/genproto"
)

// constants
const (
	port = ":50051"
	filePath = "./data/fl_checkpoint"
	chunkSize = 64 * 1024
)

// server struct to implement gRPC Round service interface 
type server struct {}


func main() {
	// listen
	lis, err := net.Listen("tcp", port)
	check(err, "Failed to listen on port" + port)

	// register FL round server
	srv := grpc.NewServer()
	pb.RegisterFlRoundServer(srv, &server {})

	// start serving
	err = srv.Serve(lis)
	check(err, "Failed to listen on port" + port)
}

// Check In rpc
// Client check in with FL server
// Server responds with FL checkpoint
func (s *server) CheckIn(stream pb.FlRound_CheckInServer) error {

	var (
		buf		[] byte
		n		int
		file	*os.File
	)

	for {
		// receive check-in request
		checkinReq, err := stream.Recv()
		log.Println("Client Name: ", checkinReq.Message)

		// open file
		file, err = os.Open(filePath)
		check(err, "Could not open new file")
		defer file.Close()

		// make a buffer of a defined chunk size
		buf = make([]byte, chunkSize)

		for {
			// read the content (by using chunks)
			n, err = file.Read(buf)
			if err == io.EOF {
				return nil
			}
			check(err, "Could not read file content")

			// send the FL Data (file chunk + type: FL checkpoint) 
			err = stream.Send(&pb.FlData{
				Message: &pb.Chunk{
					Content: buf[:n],
				},
				Type: pb.Type_FL_CHECKPOINT,
			})
		}

	}
}

// Update rpc
// Accumulate FL checkpoint update sent by client  
func (s *server) Update(stream pb.FlRound_UpdateServer) error {
	return nil
}


// Check for error, log and exit if err
func check(err error, errorMsg string) {
	if err != nil {
		log.Fatalf(errorMsg, " ==> ", err)
	}
}
