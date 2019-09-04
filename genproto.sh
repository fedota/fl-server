#!/bin/bash -e

protodir=../../fl-proto/

protoc --go_out=plugins=grpc:genproto -I $protodir $protodir/fl_round/fl_round.proto