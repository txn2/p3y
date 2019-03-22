![](mast.jpg)

# p3y: Micro Reverse Proxy

p3y was developed for use in Kubernetes, to wrap services like Prometheus with BasicAuth and TSL encryption.

## Install p3y

If you are running **MacOS** and use [homebrew] you can install **kubefwd** directly from the [txn2] tap:

```bash
# install
brew install txn2/tap/kubefwd

# ... or upgrade
brew upgrade p3y
```

## Options

| Flag          | Environment Variable | Description                                                  |
|:--------------|:---------------------|:-------------------------------------------------------------|
| -version      |                      | Display version.                                             |
| -backend      | BACKEND              | Backend server. (default "http://example.com:80")            |
| -skip-verify  | SKIP_VERIFY          | Skip backend tls verify.                                     |
| -ip           | IP                   | Server IP address to bind to. (default "0.0.0.0")            |
| -port         | PORT                 | Server port. (default "8080")                                |
| -logout       | LOGOUT               | log output stdout  (default "stdout")                        |
| -metrics_port | METRICS_PORT         | Metrics server port. (default "2112")                        |
| -username     | USERNAME             | BasicAuth username to secure Proxy.                          |
| -password     | PASSWORD             | BasicAuth password to secure Proxy.                          |
| -tls          | TLS                  | TLS Support (requires crt and key)                           |
| -crt          | CRT                  | Path to cert. (enable --tls) (default "./example.crt")       |
| -key          | KEY                  | Path to private key. (enable --tls (default "./example.key") |
| -tlsCfg       | TLSCFG               | TLS config file path.                                        |




[homebrew]:https://brew.sh/
[txn2]:https://txn2.com/