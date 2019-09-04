#!/bin/bash -e

protodir=../fl-proto/
gen=genproto

if [ ! -d $gen ]; then
    mkdir $gen;
fi

protoc --go_out=plugins=grpc:genproto -I $protodir $protodir/fl_round/fl_round.proto