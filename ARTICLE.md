# Telemetry with OpenTelemetry, Prometheus and Jaeger

## Introduction
Have you ever asked yourself "how do I know how good/bad my application is performing without waiting for breaks?". If so, this article is for you.

Today you're going to learn how to deploy an Observability Stack for your application using Kubernetes or Docker Compose.

Firstly, what's an observability stack? Basically it's a sum of components that will bring insights about an application. This is also called Telemetry.

Number of requests, average time of response of some endpoint, most used endpoints, how many errors happened during some process, and so on. These all are metrics that can be used in order to understand better how an application works and where it needs some improvements.

## Tools
There are many tools out there and, as it is with almost everything, it all depends on one needs and what one likes. 
For this article, the stack consists of OpenTelemetry, Jaeger and Prometheus.

Those tools are all Open Source, which means one can use them as he likes without worrying about licenses.

### [OpenTelemetry](https://opentelemetry.io/)
According to their website, "OpenTelemetry is a collection of tools, APIs, and SDKs. Use it to instrument, generate, collect, and export telemetry data (metrics, logs, and traces) to help you analyze your software’s performance and behavior".

So, basically, it's an aggregate of telemetry data that can be used with many different tools and libraries

### [Jaeger](https://www.jaegertracing.io/)
Jaeger it's a end-to-end distributed tracing tool that is widely used in the industry. Their [GitHub repository](https://github.com/jaegertracing/jaeger) has over 16k stars.

### [Prometheus](https://prometheus.io/)
Prometheus is a metric and alerting open source monitoring tool.

> Before we begin actually writing some code definitions, here's a disclaimer: this is probably not the best way of doing this, nor the most effective. But, this is what have worked for me and filled my demands. Feel free to leave some advices and best practices in the comments, I'll be glad.

## Architecture Overview
<div align="center">

![Image explaining how Telemetry works](/images/telemetry.png)

Obtained from: https://github.com/jaegertracing/jaeger/tree/main/docker-compose/monitor

</div>

In a "normal" stack, there's probably some application to get the tracings and another to get the metrics. That would be the case in the current scenario, but with the latest versions of Jaeger one can make use of their spans¹ to produce metrics directly from them.

Let's walkthrough the architecture:
 1) A "MicroSim" container creates simulated spans (in this case there's no MicroSim, the spans will be actually collected from the app);
 2) Then sends them to the OpenTelemetry Collector. Those spans can be instrumented manually or using some library (like `otelgin` or `otelzap`, for example);
 3) After that, the spans will be sent to the Jaeger, which will collect the tracing data, and the metrics derived from those spans will be sent to Prometheus;
 4) Finally, Jaeger will query those metrics to show both metrics and tracings in just one space.

Therefore, at the end of the article, there'll be only one dashboard (Jaeger UI) for both metrics and tracing data of the application.

## Instrumentation
As mentioned before, both metrics and tracings will derive from the same spans that one instrument.

There are many ways of instrumenting an application. It all will depend firstly on the level of control one wants over them and, secondly, which language one's using.

For `Golang`, for example, one can make use of many instrumentation libraries. If, for instance, `Gin` is used, one can get `otelgin` to instrument all HTTP requests automatically. That's precisely what the article will do.

To download `otelgin`, just run:
```bash
go get -u go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin
```

And to use it is as simple as:
```go
router = gin.New()
router.Use(otelgin.Middleware("ServiceName"))
```

The library will act as a `Gin` middleware and record the requests.

## Initiating Provider
Trace data is already being gathered, but it's not being manipulated and sent to some place. This is what the next step is.

The provider, in this case, is part of the OpenTelemetry bag of tools.
First, install its package for Go:
```bash
go get -u go.opentelemetry.io/otel
```

Basically it's needed the following things to create a provider:
- a sampler;
- a processor;
- a resource; and
- an exporter.

Here are the code snippets for them.

### Sampler
```go
import (
  "os"
  "go.opentelemetry.io/otel/sdk/trace"
)

// Helper function to define sampling.
// When in development mode, AlwaysSample is defined,
// otherwise, sample based on Parent and IDRatio will be used.
func getSampler() trace.Sampler {
	ENV := os.Getenv("GO_ENV")

	switch ENV {
	case "development":
		return trace.AlwaysSample()
	case "production":
		return trace.ParentBased(trace.TraceIDRatioBased(0.5))
	default:
		return trace.AlwaysSample()
	}
}
```
This is a help function that one can use in order to define some behaviour when in development and production mode. Sampling is, in basic terms, how often the query of data is done. So, if `trace.AlwaysSample()` is defined, for example, every single request will be traced.

The other option is the blending of two types of sampling, in which the sample will only be collected if the parent was sampled (`trace.ParentBased`) and also there's a ratio that that happens(`trace.TraceIDRatioBased(0.5)`), in this case half of cases.

### Resource
```go
import (
  "go.opentelemetry.io/otel/sdk/resource"
  semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/attribute"
)

// Returns a new OpenTelemetry resource describing this application.
func newResource(ctx context.Context) *resource.Resource {
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(os.Getenv("SERVICE_NAME")),
			attribute.String("environment", os.Getenv("GO_ENV")),
		),
	)
	if err != nil {
		log.Fatalf("%s: %v", "Failed to create resource", err)
	}

	return res
}
```
The resource is used to define an application, for one to know from where the data comes from.

### Exporter
For this, Jaeger exporter library is needed.
```bash
go get -u go.opentelemetry.io/otel/exporters/jaeger
```
In a normal environment, one would use this library with the endpoint of Jaeger in order to send the span data. But, as the span data is being sent to the OpenTelemetryCollector, the `OPEN_TELEMETRY_COLLECTOR_URL` will be used.

```go
import (
  "os"
  "go.opentelemetry.io/otel/exporters/jaeger"
)

// Creates Jaeger exporter
func exporterToJaeger() (*jaeger.Exporter, error) {
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(os.Getenv("OPEN_TELEMETRY_COLLECTOR_URL"))))
}
```

### Processor and Initiating Provider
With all this in hands, the provider can be initiated.

```go
import (
	"context"
	"io"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Initiates OpenTelemetry provider sending data to OpenTelemetry Collector.
func InitProviderWithJaegerExporter(ctx context.Context) (func(context.Context) error, error) {
	exp, err := exporterToJaeger()
	if err != nil {
		log.Fatalf("error: %s", err.Error())
	}

	tp := trace.NewTracerProvider(
		trace.WithSampler(getSampler()),
		trace.WithBatcher(exp),
		trace.WithResource(newResource(ctx)),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
```

It needs to be called before the code that will create spans, in this case, the HTTP requests with the `otelgin` library. Something like:
```go
func main() {
	ctx := context.Background()

	/// Observability
	shutdown, err := telemetry.InitProviderWithJaegerExporter(ctx)
	if err != nil {
		log.Fatalf("%s: %v", "Failed to initialize opentelemetry provider", err)
	}
	defer shutdown(ctx)

	/// rest of code
}
```

## Deploying Stack to Kubernetes and Docker Compose
At this point your application is instrumented and ready to send data to the OpenTelemetryCollector do its magic.

We'll need to deploy three services: OpenTelemetryCollector, Jaeger and Prometheus.

We can deploy it using different ways. Here are the Docker Compose and the Kubernetes way.

### Docker Compose
Docker Compose it's a great tool for everyone in the Software Development area. If one doesn't know it yet, probably it's time to learn. It helps a lot when deploying some stack in a quicker way is needed. 

> To be honest I prefer it to Kubernetes. It's more intuitive and simple.

Here's the docker-compose.yml file.
```yml
# docker-compose.yml file
version: "3.5"
services:
  jaeger:
    networks:
      - backend
    image: jaegertracing/all-in-one:latest
    volumes:
      - "./jaeger-ui.json:/etc/jaeger/jaeger-ui.json"
    command: --query.ui-config /etc/jaeger/jaeger-ui.json
    environment:
      - METRICS_STORAGE_TYPE=prometheus
      - PROMETHEUS_SERVER_URL=http://prometheus:9090
    ports:
      - "14250:14250"
      - "14268:14268"
      - "6831:6831/udp"
      - "16686:16686"
      - "16685:16685"
  otel_collector:
    networks:
      - backend
    image: otel/opentelemetry-collector-contrib:latest
    volumes:
      - "./otel-collector-config.yml:/etc/otelcol/otel-collector-config.yml"
    command: --config /etc/otelcol/otel-collector-config.yml
    ports:
      - "14278:14278"
    depends_on:
      - jaeger
  prometheus:
    networks:
      - backend
    image: prom/prometheus:latest
    volumes:
      - "./prometheus.yml:/etc/prometheus/prometheus.yml"
    ports:
      - "9090:9090"
networks:
  backend:
```

And here are the configurations one can use:
- jaeger-ui.json
```json
{
  "monitor": {
    "menuEnabled": true
  },
  "dependencies": {
    "menuEnabled": true
  }
}
```

- otel-collector-config.yml
```yml
receivers:
  jaeger:
    protocols:
      thrift_http:
        endpoint: "0.0.0.0:14278" # port opened on the otel collector

  # Dummy receiver that's never used, because a pipeline is required to have one.
  otlp/spanmetrics:
    protocols:
      grpc:
        endpoint: "localhost:65535"

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"

  otlp/jaeger: # Jaeger supports OTLP directly. The default port for OTLP/gRPC is 4317
    endpoint: "jaeger:4317"
    tls:
      insecure: true

processors:
  batch:
  spanmetrics:
    metrics_exporter: prometheus

service:
  pipelines:
    traces:
      receivers: [jaeger]
      processors: [spanmetrics, batch]
      exporters: [otlp/jaeger]
    # The exporter name in this pipeline must match the spanmetrics.metrics_exporter name.
    # The receiver is just a dummy and never used; added to pass validation requiring at least one receiver in a pipeline.
    metrics/spanmetrics:
      receivers: [otlp/spanmetrics]
      exporters: [prometheus]

```

- prometheus.yml
```yml
global:
  scrape_interval:     15s # Set the scrape interval to every 15 seconds. Default is every 1 minute.
  evaluation_interval: 15s # Evaluate rules every 15 seconds. The default is every 1 minute.
  # scrape_timeout is set to the global default (10s).

scrape_configs:
  - job_name: aggregated-trace-metrics
    static_configs:
    - targets: ['otel_collector:8889'] # using the name of the OpenTelemetryCollector container defined in the docker compose file
```

As one can see, it's pretty straightforward.

### Kubernetes
The Kubernetes version is pretty much the same as above, with only some tweaks that are necessary.

> If you find yourself at a point where you need to translate a Docker Compose file to a Kubernetes one, you can use [Kompose](https://kompose.io/), which is a conversion tool for this matter. Be aware that some tweaks will probably be necessary after the process, but it helps A LOT!

Here are the K8s files.

- jaeger.yaml
```yaml
# Namespace where the Telemetry will be created
apiVersion: v1
kind: Namespace
metadata:
  name: telemetry

---

# Jaeger ConfigMap for UI
apiVersion: v1
kind: ConfigMap
metadata:
  name: jaeger-ui-configmap
  namespace: telemetry
data:
  jaeger-ui.json: |-
    {
      "monitor": {
        "menuEnabled": true
      },
      "dependencies": {
        "menuEnabled": true
      }
    }

---

# Jaeger Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger-deployment
  namespace: telemetry
  labels:
    app: jaeger
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
    spec:
      containers:
        - args:
            - --query.ui-config
            - /etc/jaeger/ui/jaeger-ui.json
          env:
            - name: METRICS_STORAGE_TYPE
              value: prometheus
            - name: PROMETHEUS_SERVER_URL
              value: http://prometheus-service:9090
          image: jaegertracing/all-in-one:latest
          imagePullPolicy: IfNotPresent
          name: jaeger
          ports:
            - containerPort: 14250
            - containerPort: 14268
            - containerPort: 6831
              protocol: UDP
            - containerPort: 16686
            - containerPort: 16685
          volumeMounts:
            - mountPath: /etc/jaeger/ui
              name: jaeger-volume
      restartPolicy: Always
      volumes:
        - name: jaeger-volume
          configMap:
              name: jaeger-ui-configmap
              items:
                - key: jaeger-ui.json
                  path: jaeger-ui.json

---

# Jaeger Service
apiVersion: v1
kind: Service
metadata:
  name: jaeger-service
  namespace: telemetry
spec:
  selector:
    app: jaeger
  ports:
    - name: "14250"
      port: 14250
      targetPort: 14250
    - name: "14268"
      port: 14268
      targetPort: 14268
    - name: "6831"
      port: 6831
      protocol: UDP
      targetPort: 6831
    - name: "16686"
      port: 16686
      targetPort: 16686
    - name: "16685"
      port: 16685
      targetPort: 16685
```

- otel-collector.yaml
```yaml
# Otel-Collector ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  creationTimestamp: null
  name: otel-collector-configmap
  namespace: telemetry
data:
  otel-collector-config.yml: |-
    receivers:
      jaeger:
        protocols:
          thrift_http:
            endpoint: "0.0.0.0:14278" # port opened on the otel collector

      # Dummy receiver that's never used, because a pipeline is required to have one.
      otlp/spanmetrics:
        protocols:
          grpc:
            endpoint: "localhost:65535"

    exporters:
      prometheus:
        endpoint: "0.0.0.0:8889"

      otlp/jaeger: # Jaeger supports OTLP directly. The default port for OTLP/gRPC is 4317
        endpoint: "jaeger-service:4317"
        tls:
          insecure: true

    processors:
      batch:
      spanmetrics:
        metrics_exporter: prometheus

    service:
      pipelines:
        traces:
          receivers: [jaeger]
          processors: [spanmetrics, batch]
          exporters: [otlp/jaeger]
        # The exporter name in this pipeline must match the spanmetrics.metrics_exporter name.
        # The receiver is just a dummy and never used; added to pass validation requiring at least one receiver in a pipeline.
        metrics/spanmetrics:
          receivers: [otlp/spanmetrics]
          exporters: [prometheus]

---

# Otel-Collector Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otel-collector-deployment
  namespace: telemetry
  labels:
    app: otel-collector
spec:
  replicas: 1
  selector:
    matchLabels:
      app: otel-collector
  template:
    metadata:
      labels:
        app: otel-collector
    spec:
      containers:
        - args:
            - --config
            - /etc/otelcol/otel-collector-config.yml
          image: otel/opentelemetry-collector-contrib:latest
          imagePullPolicy: IfNotPresent
          name: otel-collector
          ports:
            - containerPort: 14278
            - containerPort: 8889
          volumeMounts:
            - mountPath: /etc/otelcol
              name: otel-collector-volume
      restartPolicy: Always
      volumes:
        - name: otel-collector-volume
          configMap:
              name: otel-collector-configmap
              items:
                - key: otel-collector-config.yml
                  path: otel-collector-config.yml

---

# Otel-Collector Service
apiVersion: v1
kind: Service
metadata:
  name: otel-collector-service
  namespace: telemetry
spec:
  selector:
    app: otel-collector
  ports:
    - name: "14278"
      port: 14278
      targetPort: 14278
    - name: "8889"
      port: 8889
      targetPort: 8889

```

- prometheus.yaml
```yaml
# Prometheus ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  creationTimestamp: null
  name: prometheus-configmap
  namespace: telemetry
data:
  prometheus.yml: |-
    global:
      scrape_interval:     15s # Set the scrape interval to every 15 seconds. Default is every 1 minute.
      evaluation_interval: 15s # Evaluate rules every 15 seconds. The default is every 1 minute.
      # scrape_timeout is set to the global default (10s).

    scrape_configs:
      - job_name: aggregated-trace-metrics
        static_configs:
        - targets: ['otel-collector-service:8889']

---

# Prometheus Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus-deployment
  namespace: telemetry
  labels:
    app: prometheus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
        - image: prom/prometheus:latest
          imagePullPolicy: IfNotPresent
          name: prometheus
          ports:
            - containerPort: 9090
          volumeMounts:
            - mountPath: /etc/prometheus
              name: prometheus-volume
      restartPolicy: Always
      volumes:
        - name: prometheus-volume
          configMap:
              name: prometheus-configmap
              items:
                - key: prometheus.yml
                  path: prometheus.yml

---

# Prometheus Service
apiVersion: v1
kind: Service
metadata:
  name: prometheus-service
  namespace: telemetry
spec:
  selector:
    app: prometheus
  ports:
    - name: "9090"
      port: 9090
      targetPort: 9090

```

## Summing up
As it happens we all great things, this article is coming to an end. But, before that, one may be wondering
> "What's the value of that variable OPEN_TELEMETRY_COLLECTOR_URL that we pass on to the Exporter?"

Yeah, right, that will depend in how one deployed the stack.

Basically it's the API endpoint for the traces data that the OpenTelemetryCollector will receive. It's something like:
```bash
OPEN_TELEMETRY_COLLECTOR_URL = http://{OpenTelemetryCollectorIP}:14278/api/traces
```
So, if one is, for example, running the stack using Docker Compose in the same machine as their Go application, that would be
`http://localhost:14278/api/traces`.

If one is using inside a Kubernetes cluster, that would be `http://otel-collector-service.telemetry.svc:14278/api/traces`.

That's all folks! If you have any question, just let me know.
Follow me to more subjects like this.

Cheers!

¹ Spans are "blocks of code" in the Jaeger tool that contains a context do something, like producing trace data.