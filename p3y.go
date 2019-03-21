package main

import (
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/txn2/n2proxy/sec"
	"go.uber.org/zap"
)

var Version = "0.0.0"

// BasicAuthCredentials supplied by flags
type BasicAuthCredentials struct {
	Username string
	Password string
}

// Proxy defines the Proxy handler see NewProx()
type Proxy struct {
	Target      *url.URL
	Proxy       *httputil.ReverseProxy
	Logger      *zap.Logger
	Credentials *BasicAuthCredentials
	Pmx         *Pmx
}

// Pmx
type Pmx struct {
	Requests  prometheus.Counter
	Latency   prometheus.Summary
	AuthFails prometheus.Counter
}

// handle requests
func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {

	p.Pmx.Requests.Inc()

	start := time.Now()
	reqPath := r.URL.Path
	reqMethod := r.Method

	r.Host = p.Target.Host

	p.Proxy.ServeHTTP(w, r)

	end := time.Now()
	latency := end.Sub(start)
	p.Pmx.Latency.Observe(float64(latency))

	p.Logger.Info(reqPath,
		zap.String("method", reqMethod),
		zap.String("path", reqPath),
		zap.String("time", end.Format(time.RFC3339)),
		zap.Duration("latency", latency),
	)
}

// basicAuth
func (p *Proxy) basicAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if p.Credentials == nil {
			h.ServeHTTP(w, r)
			return
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

		s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if len(s) != 2 {
			http.Error(w, "Not authorized", 401)
			p.Pmx.AuthFails.Inc()
			return
		}

		b, err := base64.StdEncoding.DecodeString(s[1])
		if err != nil {
			http.Error(w, err.Error(), 401)
			p.Pmx.AuthFails.Inc()
			return
		}

		pair := strings.SplitN(string(b), ":", 2)
		if len(pair) != 2 {
			http.Error(w, "Not authorized", 401)
			p.Pmx.AuthFails.Inc()
			return
		}

		if pair[0] != p.Credentials.Username || pair[1] != p.Credentials.Password {
			http.Error(w, "Not authorized", 401)
			p.Pmx.AuthFails.Inc()
			return
		}

		h.ServeHTTP(w, r)
	}
}

// use
func use(h http.HandlerFunc, middleware ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, m := range middleware {
		h = m(h)
	}

	return h
}

// main entry point
func main() {
	backendEnv := getEnv("BACKEND", "http://example.com:80")
	logoutEnv := getEnv("LOGOUT", "stdout")
	usernameEnv := getEnv("USERNAME", "")
	passwordEnv := getEnv("PASSWORD", "")
	portEnv := getEnv("PORT", "8080")
	ipEnv := getEnv("IP", "0.0.0.0")
	metricsPortEnv := getEnv("METRICS_PORT", "2112")

	// TLS
	crtEnv := getEnv("CRT", "./example.crt")
	keyEnv := getEnv("KEY", "./example.key")
	// skip backend TLS verification
	skpverEnvBool := false
	skpverEnv := getEnv("SKIP_VERIFY", "false")
	if skpverEnv == "true" {
		skpverEnvBool = true
	}

	tlsCfgFileEnv := getEnv("TLSCFG", "")
	tlsEnv := getEnv("TLS", "false")
	tlsEnvBool := false
	if tlsEnv == "true" {
		tlsEnvBool = true
	}

	ip := flag.String("ip", ipEnv, "Server IP address to bind to.")
	port := flag.String("port", portEnv, "Server port.")
	metricsPort := flag.String("metrics_port", metricsPortEnv, "Server port.")
	backend := flag.String("backend", backendEnv, "backend server.")
	username := flag.String("username", usernameEnv, "BasicAuth username to secure Proxy.")
	password := flag.String("password", passwordEnv, "BasicAuth password to secure Proxy.")
	srvtls := flag.Bool("tls", tlsEnvBool, "TLS Support (requires crt and key)")
	tlsCfgFile := flag.String("tlsCfg", tlsCfgFileEnv, "tls config file path.")
	crt := flag.String("crt", crtEnv, "Path to cert. (enable --tls)")
	key := flag.String("key", keyEnv, "Path to private key. (enable --tls")
	skpver := flag.Bool("skip-verify", skpverEnvBool, "Skip backend tls verify.")

	version := flag.Bool("version", false, "Display version.")
	logout := flag.String("logout", logoutEnv, "log output stdout | ")
	flag.Parse()

	if *version {
		fmt.Printf("Version: %s\n", Version)
		os.Exit(1)
	}

	zapCfg := zap.NewProductionConfig()
	zapCfg.DisableCaller = true
	zapCfg.DisableStacktrace = true
	zapCfg.OutputPaths = []string{*logout}

	logger, err := zapCfg.Build()
	if err != nil {
		fmt.Printf("Can not build logger: %s\n", err.Error())
		return
	}

	// metrics server (run in go routine)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logger.Info("Starting metrics server", zap.String("port", *metricsPort), zap.String("ip", *ip))
		err = http.ListenAndServe(*ip+":"+*metricsPort, nil)
		if err != nil {
			logger.Error("Error Starting Metrics Server", zap.Error(err))
			os.Exit(1)
		}
	}()

	logger.Info("Starting reverse proxy",
		zap.String("port", *port),
		zap.String("ip", *ip),
		zap.String("backend", *backend),
	)

	targetUrl, err := url.Parse(*backend)
	if err != nil {
		logger.Error("Unable to parse URL", zap.Error(err))
		os.Exit(1)
	}

	pxy := httputil.NewSingleHostReverseProxy(targetUrl)

	if *skpver {
		pxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	pmx := &Pmx{
		Requests: promauto.NewCounter(prometheus.CounterOpts{
			Name: "p3y_total_requests",
			Help: "Total number of requests received.",
		}),

		AuthFails: promauto.NewCounter(prometheus.CounterOpts{
			Name: "p3y_total_authentication_failures",
			Help: "Total number of authentication failures.",
		}),

		Latency: promauto.NewSummary(prometheus.SummaryOpts{
			Name: "p3y_response_time",
			Help: "Response latency.",
		}),
	}

	// Proxy
	proxy := &Proxy{
		Target: targetUrl,
		Proxy:  pxy,
		Logger: logger,
		Pmx:    pmx,
	}

	mux := http.NewServeMux()

	if *username != "" {
		proxy.Credentials = &BasicAuthCredentials{
			Username: *username,
			Password: *password,
		}
	}

	// server
	mux.HandleFunc("/", use(proxy.handle, proxy.basicAuth))

	srv := &http.Server{
		Addr:    *ip + ":" + *port,
		Handler: mux,
	}

	// If TLS is not specified serve the content unencrypted.
	if *srvtls != true {
		err = srv.ListenAndServe()
		if err != nil {
			fmt.Printf("Error starting Proxy: %s\n", err.Error())
		}
		os.Exit(0)
	}

	// Get a generic TLS configuration
	tlsCfg := sec.GenericTLSConfig()

	if *tlsCfgFile == "" {
		logger.Warn("No TLS configuration specified, using default.")
	}

	if *tlsCfgFile != "" {
		logger.Info("Loading TLS configuration from " + *tlsCfgFile)
		tlsCfg, err = sec.NewTLSCfgFromYaml(*tlsCfgFile, logger)
		if err != nil {
			fmt.Printf("Error configuring TLS: %s\n", err.Error())
			os.Exit(0)
		}
	}

	logger.Info("Starting Proxy in TLS mode.")

	srv.TLSConfig = tlsCfg
	srv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0)

	err = srv.ListenAndServeTLS(*crt, *key)
	if err != nil {
		fmt.Printf("Error starting proxyin TLS mode: %s\n", err.Error())
	}

}

// getEnv gets an environment variable or sets a default if
// one does not exist.
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}

	return value
}
