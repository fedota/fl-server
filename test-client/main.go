package main

import (
	"log"
	"context"
	"os"
	"io"
	"time"
	"sync"

	"google.golang.org/grpc"
	pb "fl-server/test-client/genproto"
)

var wg sync.WaitGroup

const (
	address = "localhost:50051"
	filePath = "./data/fl_checkpoint"
	defaultName = "Me"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	
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

	// start separate go routine to handle data dowwnload 
	go func() {

		var (
			file	*os.File
		)
		
		// open the file
		file, err = os.OpenFile(filePath, os.O_CREATE, 0644)
		check(err, "Could not open new file")
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
}

// Check for error, log and exit if err
func check(err error, errorMsg string) {
	if err != nil {
		log.Fatalf(errorMsg, " ==> ", err)
	}
}

