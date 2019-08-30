#!/bin/bash -e

protodir=../../pb

protoc --go_out=plugins=grpc:genproto -I $protodir $protodir/fl_round.proto