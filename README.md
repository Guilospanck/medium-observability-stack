# Medium Observability Stack Article
This is the example code for the Medium article "Telemetry with OpenTelemetry, Prometheus and Jaeger".

## Installation
Be sure to have Docker-Compose (or Kubernetes installed) and Golang.

## How to use
The simplest way is using Docker-Compose.
Change directory into the folder `/docker-compose` and run:
```bash
cd /docker-compose
sudo docker-compose up -d --build
```
This will deploy the three containers needed to do the Telemetry: OpenTelemetryCollector, Prometheus and Jaeger.

Then, go back to the main root and run the Golang application:
```bash
cd ..
go run .
```

Now open the Jaeger UI at `http://localhost:16686/` and run some requests to the application (`curl http://localhost:8080/ping`) in order to see the tracing data in the `Search` tab and the metrics data in the `Monitor` tab.

## Cleanup
```bash
cd /docker-compose
sudo docker-compose down
```