package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/api"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/clients"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/config"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/discovery"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/observability"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/service"
)

func main() {
	cfg := config.Load()
	podIP, err := discovery.DetectPodIP(cfg.PodIP)
	if err != nil && cfg.NacosEnabled {
		log.Fatalf("detect pod IP: %v", err)
	}
	nacos := discovery.New(cfg.NacosAddrs, cfg.StaticTargets, config.ServiceName, podIP, cfg.Port)
	if cfg.NacosEnabled {
		registerContext, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err = nacos.Register(registerContext)
		cancel()
		if err != nil {
			log.Fatalf("register with Nacos: %v", err)
		}
	}

	tracing, err := observability.New(config.ServiceName, &http.Client{})
	if err != nil {
		log.Fatalf("configure tracing: %v", err)
	}
	defer tracing.Close()
	httpClient := tracing.Client()
	routePlan := service.New(
		clients.NewTravelClient(nacos, httpClient),
		clients.NewTravel2Client(nacos, httpClient),
		clients.NewRouteClient(nacos, httpClient),
	)
	server := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Port), Handler: tracing.Wrap(api.NewRouter(routePlan))}

	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopped
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if cfg.NacosEnabled {
			if err := nacos.Deregister(shutdownContext); err != nil {
				log.Printf("deregister from Nacos: %v", err)
			}
		}
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("shutdown HTTP server: %v", err)
		}
	}()

	log.Printf("%s listening on %s", config.ServiceName, server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
