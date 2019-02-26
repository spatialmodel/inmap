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
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
	"golang.org/x/build/autocertcache"

	"github.com/spatialmodel/inmap/cloud"
	"github.com/spatialmodel/inmap/emissions/slca"
	"github.com/spatialmodel/inmap/emissions/slca/eieio"
	"github.com/spatialmodel/inmap/emissions/slca/greet"
	"github.com/spatialmodel/inmap/epi"
	"github.com/spatialmodel/inmap/inmaputil"
	"golang.org/x/crypto/acme/autocert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/testdata"
)

var (
	config     = flag.String("config", "${INMAP_ROOT_DIR}/emissions/slca/eieio/data/test_config.toml", "Path to the configuration file")
	staticRoot = flag.String("static_root", "${INMAP_ROOT_DIR}", "Path to the root directory containing InMAP source code")
	production = flag.Bool("production", false, "Is this a production setting?")
	host       = flag.String("host", "", "Address to serve from")
	tlsPort    = flag.String("tls-port", "10000", "Port to listen for encrypted requests")
	port       = flag.String("port", "8080", "Port to listen for unencrypted requests")
	bucket     = flag.String("bucket", "file://test", "Name of bucket for saving data")
)

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

func makeServerFromHandler(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
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
}

func makeHTTPServer(handler http.Handler) *http.Server {
	return makeServerFromHandler(handler)
}

func makeHTTPToHTTPSRedirectServer() *http.Server {
	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		newURI := "https://" + r.Host + r.URL.String()
		http.Redirect(w, r, newURI, http.StatusFound)
	}
	mux := &http.ServeMux{}
	mux.HandleFunc("/", handleRedirect)
	return makeServerFromHandler(mux)
}

func main() {
	flag.Parse()

	logger.Info("setting up...")
	f, err := os.Open(os.ExpandEnv(*config))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	var c eieio.ServerConfig
	_, err = toml.DecodeReader(f, &c)
	if err != nil {
		log.Fatal(err)
	}

	const eieioPrefix = "/eieio/"
	s, err := eieio.NewServer(&c, eieioPrefix, epi.NasariACS, epi.Krewski2009, epi.Krewski2009Ecologic, epi.Lepeule2012)
	if err != nil {
		logger.WithError(err).Fatal("failed to create server")
	}
	s.Log = logger

	cfg := inmaputil.InitializeConfig()

	var inmapServer *cloud.Client
	if *production {
		config, err := rest.InClusterConfig()
		if err != nil {
			logger.WithError(err).Fatal("failed to load in-cluster Kubernetes configuration")
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			logger.WithError(err).Fatal("failed to initialize Kubernetes")
		}

		inmapServer, err = cloud.NewClient(clientset, cfg.Root, cfg.Viper, *bucket, cfg.InputFiles(), cfg.OutputFiles())
		if err != nil {
			logger.WithError(err).Fatal("failed to initialize InMAP server")
		}
	} else {
		inmapServer, err = cloud.NewFakeClient(nil, nil, *bucket, cfg.Root, cfg.Viper, cfg.InputFiles(), cfg.OutputFiles())
		if err != nil {
			logger.WithError(err).Fatal("failed to initialize fake InMAP server")
		}
	}

	_, greet := initCSTDB(&s.SpatialEIO.CSTConfig)
	greet.RegisterHTTPHandlers("/greet/", filepath.Join(os.ExpandEnv(*staticRoot), "emissions", "slca"))

	mx := http.NewServeMux()
	mx.HandleFunc("/cloudrpc.CloudRPC/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r = r.WithContext(context.WithValue(ctx, "user", "default_user"))
		inmapServer.ServeHTTP(w, r)
	})
	mx.Handle("/greet/", greet)
	mx.Handle(eieioPrefix, s)
	mx.Handle("/eieiorpc.EIEIOrpc/", s)

	var m *autocert.Manager

	httpsSrv := makeHTTPServer(mx)
	httpsSrv.Addr = ":" + *tlsPort
	if *production {
		hostPolicy := func(ctx context.Context, reqHost string) error {
			if reqHost == *host || reqHost == "www."+*host {
				return nil
			}
			logger.Errorf("acme/autocert: got request for %s host but only %s or www.%s host is allowed", reqHost, *host, *host)
			return fmt.Errorf("acme/autocert: got request for %s host but only %s or www.%s host is allowed", reqHost, *host, *host)
		}

		var cache autocert.Cache
		if strings.HasPrefix(c.SpatialConfig.SpatialEIO.EIEIOCache, "gs://") {
			ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
			client, err := storage.NewClient(ctx)
			if err != nil {
				logger.Fatal(err)
			}
			loc, err := url.Parse(c.SpatialConfig.SpatialEIO.EIEIOCache)
			if err != nil {
				logger.Fatal(err)
			}
			cache = autocertcache.NewGoogleCloudStorageCache(client, loc.Host)
		} else {
			cache = autocert.DirCache(c.SpatialConfig.SpatialEIO.EIEIOCache)
		}

		m = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: hostPolicy,
			Cache:      cache,
		}
		httpsSrv.TLSConfig.GetCertificate = m.GetCertificate

		go func() {
			fmt.Printf("Starting HTTPS server on %s\n", httpsSrv.Addr)
			if err = httpsSrv.ListenAndServeTLS("", ""); err != nil {
				logger.Fatalf("httpsSrv.ListendAndServeTLS() failed with %s", err)
			}
		}()
	} else {
		go func() {
			fmt.Printf("Starting HTTPS server on %s\n", httpsSrv.Addr)
			logger.Fatal(httpsSrv.ListenAndServeTLS(testdata.Path("server1.pem"), testdata.Path("server1.key")))
		}()
	}

	httpSrv := makeHTTPToHTTPSRedirectServer()
	// allow autocert handle Let's Encrypt callbacks over http
	if m != nil {
		httpSrv.Handler = m.HTTPHandler(httpSrv.Handler)
	}

	httpSrv.Addr = ":" + *port
	fmt.Printf("Starting HTTP server on %s\n", httpSrv.Addr)
	err = httpSrv.ListenAndServe()
	if err != nil {
		log.Fatalf("httpSrv.ListenAndServe() failed with %s", err)
	}
}

// Copied from github.com/spatialmodel/inmap/emissions/slca/greet/cst_test.go
func initCSTDB(c *slca.CSTConfig) (*greet.DB, *slca.DB) {
	dir := filepath.Join(os.ExpandEnv(*staticRoot), "emissions", "slca")
	f1, err := os.Open(dir + "/greet/default.greet")
	if err != nil {
		panic(err)
	}
	f3, err := os.Open(dir + "/greet/scc/GREET to SCC.csv")
	if err != nil {
		panic(err)
	}

	f4, err := os.Open(dir + "/greet/scc/GREET vehicle SCC.csv")
	if err != nil {
		panic(err)
	}

	f5, err := os.Open(dir + "/greet/scc/GREET technology SCC.csv")
	if err != nil {
		panic(err)
	}

	lcadb := greet.Load(f1)
	if err = lcadb.AddSCCs(f3, f4, f5); err != nil {
		panic(err)
	}

	slcadb := &slca.DB{
		LCADB:     lcadb,
		CSTConfig: c,
	}
	f1.Close()
	f3.Close()
	f4.Close()
	f5.Close()

	return lcadb, slcadb
}
