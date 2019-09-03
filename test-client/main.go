package main

import (
	"log"
	"context"
	"os"
	"io"
	"time"
	"sync"
	"math/rand"

	"google.golang.org/grpc"
	pb "fl-server/test-client/genproto"
)

var wg sync.WaitGroup

const (
	address = "localhost:50051"
	filePath = "./data/fl_checkpoint"
	defaultName = "Me"
	chunkSize = 64 * 1024

)

func main() {
	// connect to the server
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	check(err, "Cannot connect to " + address)
	defer conn.Close()

	// name as input arg
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	
	// get a client
	client := pb.NewFlRoundClient(conn)
	
	// get context
	// use Minute as timeout period as server may take time to decide 
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	
	// get data 
	// ====================================================================================================
	
	// check-in with FL server
	stream, err := client.CheckIn(ctx)
	check(err, "Cannot open stream")

	// check-in ping message
	err = stream.Send(&pb.CheckInRequest {
		Message: name,
	})
	check(err, "Unable to send ping to gRPC server")

	// create channel to make program wait until the following 
	// go routine ends
	// done := make(chan bool)

	// wait for the 1 go routine to end
	wg.Add(1)

	// time.Sleep(100 * time.Millisecond)

	// start separate go routine to handle data download 
	go func() {

		var (
			file	*os.File
		)
		
		// open the file
		file, err = os.OpenFile(filePath, os.O_CREATE, 0644)
		check(err, "Could not open new checkpoint file")
		defer file.Close()

		for {
			// receive Fl data
			res, err := stream.Recv()
			// exit after data transfer completee
			if err == io.EOF {
				wg.Done()
				return
			}
			check(err, "Unable to receive data from gRPC server")
			
			// write data to file
			_, err = file.Write(res.Message.Content)
			check(err, "Unable to write into file")
		}
	} ()
	
	// <-done
	// wait 
	wg.Wait()
	// time.Sleep(300000 * time.Millisecond)

	// send data back
	// ====================================================================================================
	// check-in with FL server
	
	updateStream, err := client.Update(ctx)
	check(err, "Cannot open stream")

	file, err := os.Open(filePath)
	check(err, "Could not open checkpoint file")

	// make a buffer of a defined chunk size
	buf  := make([]byte, chunkSize)

	for {
		// read the content (by using chunks)
		n, err := file.Read(buf)
		if err == io.EOF {
			file.Close()
			break
		}
		check(err, "Could not read checkpoint file content")

		// send the FL checkpoint update Data (file chunk + type: FL checkpoint update) 
		err = updateStream.Send(&pb.FlData{
			Message: &pb.Chunk{
				Content: buf[:n],
			},
			Type: pb.Type_FL_CHECKPOINT_UPDATE,
		})
	}

	// send weight
	n := int64(rand.Intn(100))
	err = updateStream.Send(&pb.FlData{
		IntVal: n,
		Type: pb.Type_FL_CHECKPOINT_WEIGHT,
	})

	res, err := updateStream.CloseAndRecv()
	log.Println("Reconnection time: ", res.IntVal)

	// delete after sending update
	err = os.Remove(filePath)
	check(err, "Unable to delete the file")
}

// Check for error, log and exit if err
func check(err error, errorMsg string) {
	if err != nil {
		log.Fatalf(errorMsg, " ==> ", err)
	}
}

