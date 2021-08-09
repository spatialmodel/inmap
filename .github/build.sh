#!/bin/bash

cd cmd/inmap

GOOS=windows GOARCH=amd64 go build -o ../../inmap-$SOURCE_TAG-windows-amd64.exe
GOOS=linux   GOARCH=amd64 go build -o ../../inmap-$SOURCE_TAG-linux-amd64
GOOS=darwin  GOARCH=amd64 go build -o ../../inmap-$SOURCE_TAG-darwin-amd64
GOOS=darwin  GOARCH=arm64 go build -o ../../inmap-$SOURCE_TAG-darwin-arm64