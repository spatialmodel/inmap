#!/bin/bash

docker run --rm -p 8888:8888 \
  -v "$PWD":/home/jovyan/work \
  -it sr
