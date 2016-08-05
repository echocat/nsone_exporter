package main

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/prometheus/client_golang/prometheus"
	"io/ioutil"
	"github.com/prometheus/common/log"
	slog "log"
	"net/http"
)

func loadCertificatesFrom(pemFile string) (*x509.CertPool, error) {
	caCert, err := ioutil.ReadFile(pemFile)
	if err != nil {
		return nil, err
	}
	certificates := x509.NewCertPool()
	certificates.AppendCertsFromPEM(caCert)
	return certificates, nil
}

type bufferedLogWriter struct {
	buf []byte
}

func (w *bufferedLogWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func createHttpServerLogWrapper() *slog.Logger {
	return slog.New(&bufferedLogWriter{}, "", 0)
}

func startServer(metricsPath, listenAddress, tlsCert, tlsPrivateKey, tlsClientCa string) error {
	server := &http.Server{
		Addr:     listenAddress,
		ErrorLog: createHttpServerLogWrapper(),
	}
	http.Handle(metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>NSONE Exporter</title></head>
             <body>
             <h1>NSONE Exporter</h1>
             <p><a href='` + metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	if len(tlsCert) > 0 {
		clientValidation := "no"
		if len(tlsClientCa) > 0 && len(tlsCert) > 0 {
			certificates, err := loadCertificatesFrom(tlsClientCa)
			if err != nil {
				log.Fatalf("Couldn't load client CAs from %s. Got: %s", tlsClientCa, err)
			}
			server.TLSConfig = &tls.Config{
				ClientCAs:  certificates,
				ClientAuth: tls.RequireAndVerifyClientCert,
			}
			clientValidation = "yes"
		}
		targetTlsPrivateKey := tlsPrivateKey
		if len(targetTlsPrivateKey) <= 0 {
			targetTlsPrivateKey = tlsCert
		}
		log.Infof("Listening on %s (scheme=HTTPS, secured=TLS, clientValidation=%s)", listenAddress, clientValidation)
		return server.ListenAndServeTLS(tlsCert, targetTlsPrivateKey)
	} else {
		log.Infof("Listening on %s (scheme=HTTP, secured=no, clientValidation=no)", server.Addr)
		return server.ListenAndServe()
	}
}
