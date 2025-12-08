#!/bin/bash
protoc --go_out=. --go-grpc_out=. pkg/api/user/user.proto
protoc --go_out=. --go-grpc_out=. pkg/api/consent/consent.proto
protoc --go_out=. --go-grpc_out=. pkg/api/document/document.proto