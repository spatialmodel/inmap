FROM golang:1.20

WORKDIR /app

RUN git clone --depth=1 https://github.com/spatialmodel/inmap.git

WORKDIR /app/inmap

RUN go install ./...

ENV INMAP_ROOT_DIR /app/inmap/
