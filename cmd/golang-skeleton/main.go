package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.opencensus.io/tag"

	"go.opencensus.io/plugin/ochttp/propagation/b3"

	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"

	"github.com/tutabeier/golang-skeleton/pkg/health"
)

func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	initJaeger()
	pe := initPrometheus()

	srv := &http.Server{
		Addr:         ":80",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler: &ochttp.Handler{
			Handler:     initRoutes(pe),
			Propagation: &b3.HTTPFormat{},
		},
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
		CollectorEndpoint: "http://tracing:14268/api/traces",
		Process: jaeger.Process{
			ServiceName: "service",
		},
	})
	defer je.Flush()

	if err != nil {
		log.Fatalf("Failed to create the Jaeger exporter: %v", err)
	}

	trace.RegisterExporter(je)
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.ProbabilitySampler(1.0),
	})

}

func initPrometheus() *prometheus.Exporter {
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "service",
	})
	if err != nil {
		log.Fatalf("Failed to create the Prometheus exporter: %v", err)
	}

	view.RegisterExporter(pe)

	endpointTags := []tag.Key{ochttp.Method, ochttp.KeyServerRoute}

	latency := ochttp.ServerLatencyView
	latency.TagKeys = append(latency.TagKeys, endpointTags...)

	requests := ochttp.ServerRequestCountView
	requests.TagKeys = append(requests.TagKeys, endpointTags...)

	errors := ochttp.ServerResponseCountByStatusCode
	errors.TagKeys = append(errors.TagKeys, endpointTags...)

	err = view.Register(
		latency,
		requests,
		errors,
		ochttp.ServerRequestBytesView,
		ochttp.ServerResponseBytesView,
		ochttp.ClientReceivedBytesDistribution,
		ochttp.ClientSentBytesDistribution,
		ochttp.ClientRoundtripLatencyDistribution,
		ochttp.ClientCompletedCount,
		ochttp.ServerRequestCountByMethod)

	if err != nil {
		log.Print("Error registering Prometheus views")
	}

	return pe
}
