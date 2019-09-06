package main

import (
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	pb "federated-learning/fl-server/genproto/fl_round"

	"google.golang.org/grpc"
)

// constants
const (
	port					= ":50051"
	modelFilePath			= "./model/model.h5"
	checkpointFilePath		= "./data/fl_checkpoint"
	updatedCheckpointPath	= "./data/fl_checkpoint_update_"
	chunkSize				= 64 * 1024
	reconnectionTime		= 8000
	estimatedRoundTime		= 8000
	checkinLimit			= 2
	updateLimit				= 1 
	VAR_NUM_CHECKINS		= 0
	VAR_NUM_UPDATES			= 1
)

// store the result from a client
type flRoundClientResult struct {
	checkpointWeight   int64
	checkpointFilePath string
}

// to handle read writes
type readOp struct {
	varType int
	response chan int
}
type writeOp struct {
	varType int
	val int
	response chan bool
}

// server struct to implement gRPC Round service interface
type server struct {
	reads chan readOp
	writes chan writeOp
	selected chan bool
	numCheckIns       int
	numUpdates        int
	mu                sync.Mutex
	checkpointUpdates map[int]flRoundClientResult
}

func main() {
	// listen
	lis, err := net.Listen("tcp", port)
	check(err, "Failed to listen on port"+port)

	// register FL round server
	srv := grpc.NewServer()
	// impl instance
	flServer := &server{numCheckIns: 0, checkpointUpdates: make(map[int]flRoundClientResult), reads: make(chan readOp), writes: make(chan writeOp)}
	pb.RegisterFlRoundServer(srv, flServer)

	// go flServer.EventLoop()
	go flServer.ConnectionMetrics()

	// start serving
	err = srv.Serve(lis)
	check(err, "Failed to serve on port " + port)
}

// Check In rpc
// Client check in with FL server
// Server responds with FL checkpoint
func (s *server) CheckIn(stream pb.FlRound_CheckInServer) error {

	var (
		buf  []byte
		n    int
		file *os.File
	)

	// receive check-in request
	checkinReq, err := stream.Recv()
	log.Println("Client Name: ", checkinReq.Message)

	// // prevent inconsistency
	// // as each rpc is executed as a separate go routine
	// s.mu.Lock()
	// s.numCheckIns++
	// s.mu.Unlock()
	// log.Println("Count: ", s.numCheckIns)
	
	// create a write operation
	write := writeOp{
		varType:  VAR_NUM_CHECKINS,
		response: make(chan bool)}
	// send to handler(ConnectionMetrics) via writes channel
	s.writes <- write
	
	// wait for response
	if !(<- write.response) {
		log.Println("CheckIn rejected")
		return nil
	}

	// wait for selection
	return nil

	// open file
	file, err = os.Open(checkpointFilePath)
	check(err, "Could not open checkpoint file")
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

	// // count number of operation
	// // as each rpc is executed as a separate go routine
	// // TODO: make go routine to handle count updates
	// s.mu.Lock()
	// s.numUpdates++
	// index := s.numUpdates
	// s.mu.Unlock()

	// create a write operation
	write := writeOp{
		varType:  VAR_NUM_UPDATES,
		response: make(chan bool)}
	// send to handler (ConnectionMetrics) via writes channel
	s.writes <- write
	
	// wait for response
	if !(<- write.response) {
		log.Println("Update rejected")
		return nil
	}

	// create read operation
	read := readOp{
		varType:  VAR_NUM_UPDATES,
		response: make(chan int)}
	// send to handler (ConnectionMetrics) via reads channel
		s.reads <- read

	index := <- read.response

	// open the file
	// log.Println(updatedCheckpointPath + strconv.Itoa(index))
	filePath := updatedCheckpointPath + strconv.Itoa(index)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, os.ModeAppend)
	check(err, "Could not open new checkpoint file")
	defer file.Close()

	for {
		// receive Fl data
		flData, err := stream.Recv()
		// exit after data transfer completee
		if err == io.EOF {
			return stream.SendAndClose(&pb.FlData{
				IntVal: reconnectionTime,
				Type:   pb.Type_FL_RECONN_TIME,
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
			s.checkpointUpdates[index] = flRoundClientResult{
				checkpointWeight:   flData.IntVal,
				checkpointFilePath: filePath,
			}
			s.mu.Unlock()
			log.Println("Checkpoint Update: ", s.checkpointUpdates[index])
		}
	}
}

func (s *server) EventLoop() {
	// TODO: change this naive way
	// Wait for an estimated time for round
	log.Println("Event Loop in sleep")
	time.Sleep(estimatedRoundTime * time.Millisecond)
	log.Println("Event loop out of sleep")

	s.FederatedAveraging()

	log.Println("Federated Averaging completed")

	// or compare with some base number
	// TODO: change else condition
	// if s.numUpdates == s.numCheckIns && s.numUpdates > 0 {
	// 	// assumed no more updates and check-ins
	// 	s.FederatedAveraging()
	// } else {
	// 	log.Fatalf("Could not complete round")
	// }
}

// Runs federated averaging
func (s *server) FederatedAveraging() {
	// determine arguments
	args := ""
	for _, v := range s.checkpointUpdates {
		args += strconv.FormatInt(v.checkpointWeight, 10) + " " + v.checkpointFilePath + " "
	}

	argsList := []string{"federated_averaging.py", "--cf", checkpointFilePath, "--mf", modelFilePath, "--u", "281", "./device1/weight_updates", "281", "./device2/weight_updates", "296", "./device3/weight_updates"}
	// if args == "" {
	// 	return
	// }

	// model path
	cmd := exec.Command("python", argsList...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	check(err, "Unable to run federated averaging")
}

// Handler for connection reads and updates
// Takes care of update and checkin limits 
func (s *server) ConnectionMetrics() {
	for {
		select {
		// read query
		case read := <- s.reads:
			switch read.varType {
			case VAR_NUM_CHECKINS:
				read.response <- s.numCheckIns
			case VAR_NUM_UPDATES:
				read.response <- s.numUpdates
			}
		// write query
		case write := <- s.writes:
			switch write.varType {
			case VAR_NUM_CHECKINS:
				s.numCheckIns++
				// if number of checkins exceed the limit, reject 
				if (s.numCheckIns > checkinLimit) {
					write.response <- false
				} else {
					write.response <- true
				}
			case VAR_NUM_UPDATES:
				s.numUpdates++
				// if enough updates available, reject
				if (s.numUpdates > updateLimit) {
					write.response <- false
					// begin federated averaging process 
					// go s.FederatedAveraging()
					log.Println("FA Process")
				} else {
					write.response <- true
				}
			}
		}
	}
}

// Check for error, log and exit if err
func check(err error, errorMsg string) {
	if err != nil {
		log.Fatalf(errorMsg+" ==> ", err)
	}
}
