#!/bin/bash

# This script compiles InMAP for different systems.

version=1.4.1

env GOOS=linux GOARCH=amd64 go build -v
mv inmap inmap${version}linux-amd64

env GOOS=darwin GOARCH=amd64 go build -v
mv inmap inmap${version}darwin-amd64

env GOOS=windows GOARCH=amd64 go build -v
mv inmap.exe inmap${version}windows-amd64.exe
