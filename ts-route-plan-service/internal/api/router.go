package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/domain"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/security"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/service"
)

const basePath = "/api/v1/routeplanservice"

func NewRouter(routePlan *service.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+basePath+"/welcome", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("Welcome to [ RoutePlan Service ] !"))
	})
	mux.HandleFunc("POST "+basePath+"/routePlan/cheapestRoute", jsonRoute(routePlan.Cheapest))
	mux.HandleFunc("POST "+basePath+"/routePlan/quickestRoute", jsonRoute(routePlan.Quickest))
	mux.HandleFunc("POST "+basePath+"/routePlan/minStopStations", jsonRoute(routePlan.MinStops))
	return cors(jwt(mux))
}

func jsonRoute(operation func(context.Context, domain.RoutePlanInfo) (domain.Response[[]domain.RoutePlanResultUnit], error)) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		var info domain.RoutePlanInfo
		decoder := json.NewDecoder(request.Body)
		if err := decoder.Decode(&info); err != nil {
			http.Error(writer, "", http.StatusBadRequest)
			return
		}
		response, err := operation(request.Context(), info)
		if err != nil {
			http.Error(writer, "", http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}
}

func jwt(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := security.ValidateBearer(request.Header.Get("Authorization"), time.Now()); err != nil {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin != "" {
			writer.Header().Set("Access-Control-Allow-Origin", "*")
			writer.Header().Add("Vary", "Origin")
		}
		if request.Method == http.MethodOptions && request.Header.Get("Access-Control-Request-Method") != "" {
			writer.Header().Set("Access-Control-Allow-Methods", request.Header.Get("Access-Control-Request-Method"))
			if requestedHeaders := request.Header.Get("Access-Control-Request-Headers"); requestedHeaders != "" {
				writer.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
			}
			writer.Header().Set("Access-Control-Max-Age", "3600")
			writer.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(writer, request)
	})
}
