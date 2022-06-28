# Telemetry
We are using `OpenTelemetry` + `Jaeger` + `Prometheus` as stack of Observability.
See https://github.com/jaegertracing/jaeger/tree/main/docker-compose/monitor for more details.

Using this stack, we can have `tracings` producing automatically some `metrics` for us.

## How to use (dev)
```bash
sudo docker-compose up -d --build
```
Now you can go to:
- `http://localhost:16686` for the Jaeger UI; or
- `http://localhost:9090` for the Prometheus UI.
