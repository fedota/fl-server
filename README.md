# Federated Learning Server

### Usage

1. Start Server
	```
	$ cd server
	$ chmod +x genproto.sh && ./genproto
	$ go run main.go
	```

1. Start a Test Client to connect to the server
	```
	$ cd test-client
	$ chmod +x genproto.sh && ./genproto
	$ go run main.go <Name-of-client>
	```

### Compile proto
```
$ protoc --go_out=plugins=grpc:server/genproto -I pb pb/fl_round.proto
$ protoc --go_out=plugins=grpc:test-client/genproto -I pb pb/fl_round.proto
```
