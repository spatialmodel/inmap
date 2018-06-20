FROM golang

WORKDIR /go/src/github.com/spatialmodel

RUN git clone --depth=1 https://github.com/spatialmodel/inmap.git

WORKDIR /go/src/github.com/spatialmodel/inmap

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN dep ensure

RUN go install github.com/spatialmodel/inmap/cmd/...

# This step installs ssl certificates for testing.
RUN go get google.golang.org/grpc/testdata

EXPOSE 10000
EXPOSE 8080
