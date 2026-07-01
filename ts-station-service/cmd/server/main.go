package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	_ "github.com/go-sql-driver/mysql"

	"github.com/FudanSELab/train-ticket/ts-station-service/internal/api"
	"github.com/FudanSELab/train-ticket/ts-station-service/internal/config"
	"github.com/FudanSELab/train-ticket/ts-station-service/internal/discovery"
	"github.com/FudanSELab/train-ticket/ts-station-service/internal/observability"
	"github.com/FudanSELab/train-ticket/ts-station-service/internal/repository"
	"github.com/FudanSELab/train-ticket/ts-station-service/internal/service"
)

func main() {
	cfg := config.Load()
	database, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	startupContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.PingContext(startupContext); err != nil {
		log.Fatalf("connect to station MySQL: %v", err)
	}
	repo := repository.NewMySQL(database)
	if err := repo.EnsureSchema(startupContext); err != nil {
		log.Fatalf("ensure station schema: %v", err)
	}
	stationService := service.New(repo)
	if err := stationService.Seed(startupContext); err != nil {
		log.Fatalf("seed stations: %v", err)
	}

	podIP, err := discovery.DetectPodIP(cfg.PodIP)
	if err != nil && cfg.NacosEnabled {
		log.Fatalf("detect pod IP: %v", err)
	}
	registrar := discovery.NewRegistrar(cfg.NacosAddrs, config.ServiceName, podIP, cfg.Port)
	if cfg.NacosEnabled {
		if err := registrar.Register(startupContext); err != nil {
			log.Fatalf("register with Nacos: %v", err)
		}
	}

	tracing, err := observability.New(config.ServiceName)
	if err != nil {
		log.Fatalf("configure tracing: %v", err)
	}
	defer tracing.Close()
	server := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Port), Handler: tracing.Wrap(api.NewRouter(stationService))}
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopped
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if cfg.NacosEnabled {
			if err := registrar.Deregister(ctx); err != nil {
				log.Printf("deregister from Nacos: %v", err)
			}
		}
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("shutdown HTTP server: %v", err)
		}
	}()
	log.Printf("%s listening on %s", config.ServiceName, server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
