package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/FudanSELab/train-ticket/ts-station-service/internal/domain"
	"github.com/FudanSELab/train-ticket/ts-station-service/internal/security"
	"github.com/FudanSELab/train-ticket/ts-station-service/internal/service"
)

const basePath = "/api/v1/stationservice"

func NewRouter(stations *service.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+basePath+"/welcome", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("Welcome to [ Station Service ] !"))
	})
	mux.HandleFunc("GET "+basePath+"/stations", responseHandler(http.StatusOK, func(request *http.Request) (domain.Response, error) { return stations.Query(request.Context()) }))
	mux.HandleFunc("POST "+basePath+"/stations", func(writer http.ResponseWriter, request *http.Request) {
		var station domain.Station
		if err := decode(request, &station); err != nil || station.Name == nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		result, err := stations.Create(request.Context(), station)
		writeOperation(writer, http.StatusCreated, result, err)
	})
	mux.HandleFunc("PUT "+basePath+"/stations", func(writer http.ResponseWriter, request *http.Request) {
		var station domain.Station
		if err := decode(request, &station); err != nil || station.Name == nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		result, err := stations.Update(request.Context(), station)
		writeOperation(writer, http.StatusOK, result, err)
	})
	mux.HandleFunc("DELETE "+basePath+"/stations/{stationsId}", responseHandler(http.StatusOK, func(request *http.Request) (domain.Response, error) {
		return stations.Delete(request.Context(), request.PathValue("stationsId"))
	}))
	mux.HandleFunc("GET "+basePath+"/stations/id/{stationNameForId}", responseHandler(http.StatusOK, func(request *http.Request) (domain.Response, error) {
		return stations.QueryForID(request.Context(), request.PathValue("stationNameForId"))
	}))
	mux.HandleFunc("POST "+basePath+"/stations/idlist", func(writer http.ResponseWriter, request *http.Request) {
		var names []string
		if err := decode(request, &names); err != nil || names == nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		result, err := stations.QueryForIDBatch(request.Context(), names)
		writeOperation(writer, http.StatusOK, result, err)
	})
	mux.HandleFunc("GET "+basePath+"/stations/name/{stationIdForName}", responseHandler(http.StatusOK, func(request *http.Request) (domain.Response, error) {
		return stations.QueryByID(request.Context(), request.PathValue("stationIdForName"))
	}))
	mux.HandleFunc("POST "+basePath+"/stations/namelist", func(writer http.ResponseWriter, request *http.Request) {
		var ids []string
		if err := decode(request, &ids); err != nil || ids == nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		result, err := stations.QueryByIDBatch(request.Context(), ids)
		writeOperation(writer, http.StatusOK, result, err)
	})
	return cors(authorize(mux))
}

func responseHandler(status int, operation func(*http.Request) (domain.Response, error)) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		result, err := operation(request)
		writeOperation(writer, status, result, err)
	}
}

func writeOperation(writer http.ResponseWriter, status int, response domain.Response, err error) {
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(response)
}

func decode(request *http.Request, output any) error {
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(output); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		auth, err := security.ParseOptionalBearer(request.Header.Get("Authorization"), time.Now())
		if err != nil {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		// This exact path check deliberately preserves the Java antMatcher quirk:
		// DELETE /stations/{id} is public because the protected matcher is /stations.
		if (request.Method == http.MethodPost || request.Method == http.MethodPut || request.Method == http.MethodDelete) && request.URL.Path == basePath+"/stations" {
			if !auth.Present || !auth.Roles["ROLE_ADMIN"] {
				writer.WriteHeader(http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(writer, request)
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Origin") != "" {
			writer.Header().Set("Access-Control-Allow-Origin", "*")
			writer.Header().Add("Vary", "Origin")
		}
		if request.Method == http.MethodOptions && request.Header.Get("Access-Control-Request-Method") != "" {
			writer.Header().Set("Access-Control-Allow-Methods", request.Header.Get("Access-Control-Request-Method"))
			if requested := request.Header.Get("Access-Control-Request-Headers"); requested != "" {
				writer.Header().Set("Access-Control-Allow-Headers", requested)
			}
			writer.Header().Set("Access-Control-Max-Age", "3600")
			writer.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(writer, request)
	})
}
