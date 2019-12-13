package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/tutabeier/golang-skeleton/pkg/config"
	"github.com/tutabeier/golang-skeleton/pkg/health"
	"github.com/tutabeier/golang-skeleton/pkg/users"

	"contrib.go.opencensus.io/exporter/jaeger"
	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/opencensus-integrations/ocsql"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

const databaseRetries = 5

func main() {
	env := config.GetEnv()

	driverName, err := ocsql.Register("postgres", ocsql.WithAllTraceOptions())
	if err != nil {
		log.Panicf("Unable to register OCSQL: %s", err.Error())
	}
	db, err := sql.Open(driverName, env.DatabaseDSN)
	if err != nil {
		log.Panicf("Error opening database: %s", err.Error())
	}
	defer db.Close()

	var i time.Duration
	for {
		if i == databaseRetries {
			log.Panicf("Exceed retries. DB is not ready.")
		}

		err = db.Ping()
		if err == nil {
			log.Print("Postgres ready.")
			break
		}

		log.Print(err.Error())

		time.Sleep(i * time.Second)
		log.Printf("Waiting for Postgres to become ready. Trying again in %ds.", i)
		i++
	}

	// TODO: add semaphore to avoid multiple instances running the migrations
	log.Print("Start running migrations")
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	m, err := migrate.NewWithDatabaseInstance(
		env.MigrationsFolder,
		"postgres", driver)
	if err != nil {
		log.Fatalf("Unable to run migrations: %s", err.Error())
	}
	m.Steps(2)
	log.Print("Finished running migrations")

	ur := users.NewRepository(db)
	uh := users.NewHandler(ur)

	r := http.NewServeMux()
	instrumentedRoute(r, "/users", uh.Handle())

	r.Handle("/metrics", initPrometheus())
	r.Handle("/status", health.Check())

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", env.Port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler: &ochttp.Handler{
			Handler:     r,
			Propagation: &b3.HTTPFormat{},
		},
	}

	go func() {
		log.Printf("Running server at %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
	go initJaeger(env.JaegerHost)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err = srv.Shutdown(ctx)
	if err != nil {
		log.Fatalf("Error when sutting down server %s", err.Error())
	}
	log.Println("Sutting down")
	os.Exit(0)
}

func initJaeger(host string) {
	je, err := jaeger.NewExporter(jaeger.Options{
		CollectorEndpoint: fmt.Sprintf("http://%s:14268/api/traces", host),
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

	endpointTags := []tag.Key{ochttp.Method, ochttp.Path}

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
		log.Printf("Error registering Prometheus views: %s", err.Error())
	}

	return pe
}

func instrumentedRoute(r *http.ServeMux, route string, handler http.Handler) {
	withRequestID := func(w http.ResponseWriter, r *http.Request) {
		span := trace.FromContext(r.Context())
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			id, _ := uuid.NewRandom()
			requestID = id.String()
		}

		span.AddAttributes(trace.StringAttribute("request.id", requestID))
		ctx := trace.NewContext(r.Context(), span)
		handler.ServeHTTP(w, r.WithContext(ctx))
	}

	h := &ochttp.Handler{
		Handler: http.HandlerFunc(withRequestID),
		FormatSpanName: func(r *http.Request) string {
			return fmt.Sprintf(" %s %s", r.Method, route)
		},
	}

	r.Handle(route, ochttp.WithRouteTag(h, route))
}
