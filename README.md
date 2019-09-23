# Federated Learning Server

### Usage

* Refer [fl-misc](https://github.com/ShashankP19/fl-misc) for setup instructions
* `Server` folder contains
	* `model/model.h5`: Model
	* `checkpoint/fl_checkpoint`: Checkpoint
	* `weight_updates`: Client updates stored here

1. Start Server
	```
	cd fl-server
	chmod +x genproto.sh && ./genproto
	go run main.go
	```

	Using Docker
	```
	cd fl-server
	chmod +x genproto.sh && ./genproto
	docker build -t fl-server .
	docker run -d -p 50051:50051 fl-server
	```

2. Start a Test Client to connect to the server
	```
	cd fl-misc/test-client
	chmod +x genproto.sh && ./genproto
	go run main.go <Name-of-client>
	```
