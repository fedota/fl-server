package main

import (
	"log"
	"net"
	"os"
	"io"
	"sync"
	"strconv"
	"time"
	"os/exec"

	"google.golang.org/grpc"
	pb "fl-server/server/genproto"
)

// constants
const (
	port = ":50051"
	checkpointFilePath = "./data/fl_checkpoint"
	updatedCheckpointPath = "./data/fl_checkpoint_update_"
	chunkSize = 64 * 1024
	reconnectionTime = 8000
	estimatedRoundTime = 1000
)


// server struct to implement gRPC Round service interface 
type server struct {
	numCheckIns int
	numUpdates int
	mu  sync.Mutex
	checkpointUpdates map[int]int64
}


func main() {
	// listen
	lis, err := net.Listen("tcp", port)
	check(err, "Failed to listen on port" + port)

	// register FL round server
	srv := grpc.NewServer()
	// impl instance
	flServer := &server {numCheckIns: 0, checkpointUpdates: make(map[int]int64)}
	pb.RegisterFlRoundServer(srv, flServer)

	// start serving
	err = srv.Serve(lis)
	check(err, "Failed to serve on port" + port)

	// go flServer.EventLoop()
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
	
	// receive check-in request
	checkinReq, err := stream.Recv()
	log.Println("Client Name: ", checkinReq.Message)

	// prevent inconsistency
	// as each rpc is executed as a separate go routine
	s.mu.Lock()
	s.numCheckIns++
	s.mu.Unlock()
	log.Println("Count: ", s.numCheckIns)
	
	// open file
	file, err = os.Open(checkpointFilePath)
	check(err, "Could not open checkpoint file")
	defer file.Close()

	// make a buffer of a defined chunk size
	buf  = make([]byte, chunkSize)

	for {
		// read the content (by using chunks)
		n, err = file.Read(buf)
		if err == io.EOF {
			return nil
		}
		check(err, "Could not read file content")

		// send the FL checkpoint Data (file chunk + type: FL checkpoint) 
		err = stream.Send(&pb.FlData{
			Message: &pb.Chunk{
				Content: buf[:n],
			},
			Type: pb.Type_FL_CHECKPOINT,
		})
	}

}

// Update rpc
// Accumulate FL checkpoint update sent by client
// TODO: delete file when error
func (s *server) Update(stream pb.FlRound_UpdateServer) error {

	// count number of operation
	// as each rpc is executed as a separate go routine
	// TODO: make go routine to handle count updates
	s.mu.Lock()
	s.numUpdates++
	index := s.numUpdates
	s.mu.Unlock()

	// open the file
	// log.Println(updatedCheckpointPath + strconv.Itoa(index))
	file, err := os.OpenFile(updatedCheckpointPath + strconv.Itoa(index), os.O_CREATE, 0644)
	check(err, "Could not open new checkpoint file")
	defer file.Close()

	for {
		// receive Fl data
		flData, err := stream.Recv()
		// exit after data transfer completee
		if err == io.EOF {
			return stream.SendAndClose(&pb.FlData{
				IntVal: reconnectionTime,
				Type: pb.Type_FL_RECONN_TIME,
			})
		}
		check(err, "Unable to receive update data from client")
		
		if flData.Type == pb.Type_FL_CHECKPOINT_UPDATE {
			// write data to file
			_, err = file.Write(flData.Message.Content)
			check(err, "Unable to write into new checkpoint file")
		} else if flData.Type == pb.Type_FL_CHECKPOINT_WEIGHT {
			// put in array
			// TODO: modify by making a go-routine to do updates
			s.mu.Lock()
			s.checkpointUpdates[index] = flData.IntVal
			s.mu.Unlock()
			log.Println("Checkpoint Update: ", s.checkpointUpdates[index])
		}
	}
}

func (s *server) EventLoop() {
	// TODO: change this naive way
	// Wait for an estimated time for round 
	time.Sleep(estimatedRoundTime * time.Millisecond)

	// if s.numUpdates == come fixed amount or s.numCheckIns
	s.FederatedAveraging()
}

// launch the fede
func (s *server) FederatedAveraging() {
	// file paths and model path
	err := exec.Command("federated_averaging.py", "./data/fl_checkpoint", "./data", "").Run()
	check(err, "Unable to run federated averaging")
}

// Check for error, log and exit if err
func check(err error, errorMsg string) {
	if err != nil {
		log.Fatalf(errorMsg, " ==> ", err)
	}
}
