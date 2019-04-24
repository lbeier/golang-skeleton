package main

import (
	"context"
	"flag"
	"go.opencensus.io/tag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/tutabeier/golang-skeleton/pkg/health"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	initJaeger()
	pe := initPrometheus()
	registerViews()

	srv := &http.Server{
		Addr:         ":9999",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      initRoutes(pe),
	}

	go func() {
		log.Printf("Running server at %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("shutting down")
	os.Exit(0)
}

func initRoutes(p *prometheus.Exporter) *http.ServeMux {
	r := http.NewServeMux()

	r.HandleFunc("/status", health.Check())
	r.Handle("/metrics", p)

	return r
}

func initJaeger() {
	je, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: "127.0.0.1:6831",
		Process: jaeger.Process{
			ServiceName: "golang-skeleton",
		},
	})

	if err != nil {
		log.Fatalf("Failed to create the Jaeger exporter: %v", err)
	}

	trace.RegisterExporter(je)
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.ProbabilitySampler(1.0),
	})
}

func initPrometheus() *prometheus.Exporter{
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "golang-skeleton",
	})
	if err != nil {
		log.Fatalf("Failed to create the Prometheus exporter: %v", err)
	}

	view.RegisterExporter(pe)

	return pe
}

func registerViews() {
	endpointTags := []tag.Key{ochttp.Method, ochttp.KeyServerRoute}

	latency := ochttp.ServerLatencyView
	latency.TagKeys = append(latency.TagKeys, endpointTags...)

	requests := ochttp.ServerRequestCountView
	requests.TagKeys = append(requests.TagKeys, endpointTags...)

	errors := ochttp.ServerResponseCountByStatusCode
	errors.TagKeys = append(errors.TagKeys, endpointTags...)

	view.Register(
		latency,
		requests,
		errors,
		ochttp.ServerRequestBytesView,
		ochttp.ServerResponseBytesView,
		ochttp.ClientReceivedBytesDistribution,
		ochttp.ClientSentBytesDistribution,
		ochttp.ClientRoundtripLatencyDistribution,
		ochttp.ClientCompletedCount,
	)
}