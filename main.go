package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Guilospanck/medium-observability-stack/telemetry"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	ctx := context.Background()

	/// Observability
	shutdown, err := telemetry.InitProviderWithJaegerExporter(ctx)
	if err != nil {
		log.Fatalf("%s: %v", "Failed to initialize opentelemetry provider", err)
	}
	defer shutdown(ctx)

	// Gin routes using otelgin
	r := gin.Default()
	r.Use(otelgin.Middleware("OBSERVABILITY-MEDIUM-SERVICE-EXAMPLE"))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
