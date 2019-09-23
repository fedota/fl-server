FROM golang:latest

# Install dependencies
RUN apt-get update && \
	apt-get install -y python3-dev python3-pip
RUN pip3 install --upgrade numpy tensorflow keras

RUN go get -u google.golang.org/grpc
RUN go get -u github.com/golang/protobuf/proto 

ENV GO_SERVER_PATH federated-learning/fl-server
ADD . /go/src/$GO_SERVER_PATH
WORKDIR /go/src/$GO_SERVER_PATH
# RUN go install $GO_SERVER_PATH

ENTRYPOINT ["go", "run", "main.go"]
# /go/bin/fl-server

EXPOSE 50051