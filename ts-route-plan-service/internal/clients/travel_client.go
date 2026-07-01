package clients

import (
	"context"
	"net/http"
	"net/url"

	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/discovery"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/domain"
)

const travelServiceName = "ts-travel-service"

type TravelClient struct{ transport transport }

func NewTravelClient(resolver discovery.Resolver, client *http.Client) *TravelClient {
	return &TravelClient{transport: newTransport(resolver, client)}
}

func (c *TravelClient) LeftTrips(ctx context.Context, info domain.TripInfo) ([]domain.TripResponse, error) {
	var response domain.Response[[]domain.TripResponse]
	err := c.transport.exchange(ctx, travelServiceName, http.MethodPost, "/api/v1/travelservice/trips/left", info, &response)
	return response.Data, err
}

func (c *TravelClient) RouteForTrip(ctx context.Context, tripID string) (*domain.Route, error) {
	var response domain.Response[*domain.Route]
	err := c.transport.exchange(ctx, travelServiceName, http.MethodGet, "/api/v1/travelservice/routes/"+url.PathEscape(tripID), nil, &response)
	return response.Data, err
}

func (c *TravelClient) TripsByRoutes(ctx context.Context, routeIDs []string) ([][]domain.Trip, error) {
	var response domain.Response[[][]domain.Trip]
	err := c.transport.exchange(ctx, travelServiceName, http.MethodPost, "/api/v1/travelservice/trips/routes", routeIDs, &response)
	return response.Data, err
}

func (c *TravelClient) TripDetail(ctx context.Context, info domain.TripAllDetailInfo) (*domain.TripAllDetail, error) {
	var response domain.Response[*domain.TripAllDetail]
	err := c.transport.exchange(ctx, travelServiceName, http.MethodPost, "/api/v1/travelservice/trip_detail", info, &response)
	return response.Data, err
}
