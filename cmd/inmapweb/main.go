/*
Copyright © 2018 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

// command inmapweb is a web interface for the InMAP model and related tools.
package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
	"github.com/spatialmodel/inmap/emissions/slca/bea/eioserve"

	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/testdata"
)

const Address = ":10000"

var logger *logrus.Logger

func init() {
	logger = logrus.StandardLogger()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
		DisableSorting:  true,
	})
	// Should only be done from init functions
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(logger.Out, logger.Out, logger.Out))
}

var config = flag.String("config", "${GOPATH}/src/github.com/spatialmodel/inmap/emissions/slca/bea/data/example_config.toml", "Path to the configuration file")

func main() {
	flag.Parse()

	f, err := os.Open(os.ExpandEnv(*config))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	var c eioserve.ServerConfig
	_, err = toml.DecodeReader(f, &c)
	if err != nil {
		log.Fatal(err)
	}

	logger.Info("setting up...")
	s, err := eioserve.NewServer(&c)
	if err != nil {
		logger.WithError(err).Fatal("failed to create server")
	}
	s.Log = logger

	srv := &http.Server{
		Addr:              Address,
		Handler:           s,
		ReadHeaderTimeout: 5 * time.Second,
		//WriteTimeout: 5 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSConfig: &tls.Config{
			PreferServerCipherSuites: true,
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519,
			},
		},
	}

	logger.Infof("listening on https://%s\n", Address)
	logger.Fatal(srv.ListenAndServeTLS(testdata.Path("server1.pem"), testdata.Path("server1.key")))
}
