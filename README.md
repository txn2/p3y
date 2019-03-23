![](mast.jpg)
[![](https://images.microbadger.com/badges/image/txn2/p3y.svg)](https://microbadger.com/images/txn2/p3y "p3y")

# p3y: Micro Reverse Proxy

p3y is a small single binary reverse proxy written in go and was developed
for use in Kubernetes, to wrap services like Prometheus with BasicAuth
and TSL encryption. p3y exposes its operational metrics on port 2112 by default.

## Quick Docker Example

Proxy your local port **8080** to site https://example.com.

```bash
docker run --rm -p 8080:8080 -p 2112:2112 txn2/p3y \
    -backend https://example.com:443 \
    -username test \
    -password test
```

Open http://localhost:8080 to view the site or http://localhost:2112 to view metrics.


## Install p3y on a Mac

If you are running **MacOS** and use [homebrew] you can install **kubefwd** directly from the [txn2] tap:

```bash
# install
brew install txn2/tap/kubefwd

# ... or upgrade
brew upgrade p3y
```

## CLI Options

| Flag          | Environment Variable | Description                                                  |
|:--------------|:---------------------|:-------------------------------------------------------------|
| -help         |                      | Display help.                                                |
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

## Kubernetes Example

The following sets up two services, one for the p3y proxy exposed on **NodePort 30090**, this should now
be accessible from outside the cluster. Metrics for the proxy are available inside the cluster at
**http://prom-proxy-metrics:2112/metrics.

Example Services:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: prom-proxy
  namespace: example
  labels:
    app: prom-proxy
spec:
  selector:
    app: prom-proxy
  ports:
    - protocol: "TCP"
      port: 9090
      nodePort: 30090
      targetPort: 9090
  type: NodePort
---
apiVersion: v1
kind: Service
metadata:
  name: prom-proxy-metrics
  namespace: example
  labels:
    app: prom-proxy
spec:
  selector:
    app: prom-proxy
  ports:
    - protocol: "TCP"
      port: 80
      targetPort: 2112
  type: ClusterIP
```

Example Deployment:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prom-proxy
  namespace: example
  labels:
    app: prom-proxy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prom-proxy
  template:
    metadata:
      labels:
        app: prom-proxy
        component: idx
    spec:
      containers:
        - name: prom-proxy
          image: txn2/p3y:1.0.0
          imagePullPolicy: IfNotPresent
          args: [
            "-port=9090",
            "-backend=http://prometheus:9090",
            "-username=somebody",
            "-password=goodlongpassword",
            "-tls",
            "-crt=/cert/server.crt",
            "-key=/cert/server.key"
          ]
          ports:
            - name: http
              containerPort: 9090
            - name: metrics
              containerPort: 2112
          volumeMounts:
            - name: prom-proxy-cert
              mountPath: "/cert"
      volumes:
        - name: prom-proxy-cert
          secret:
            secretName: prom-proxy-cert
```


[homebrew]:https://brew.sh/
[txn2]:https://txn2.com/