package clients

import (
	"context"
	"net/http"
	"net/url"

	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/discovery"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/domain"
)

const travel2ServiceName = "ts-travel2-service"

type Travel2Client struct{ transport transport }

func NewTravel2Client(resolver discovery.Resolver, client *http.Client) *Travel2Client {
	return &Travel2Client{transport: newTransport(resolver, client)}
}

func (c *Travel2Client) LeftTrips(ctx context.Context, info domain.TripInfo) ([]domain.TripResponse, error) {
	var response domain.Response[[]domain.TripResponse]
	err := c.transport.exchange(ctx, travel2ServiceName, http.MethodPost, "/api/v1/travel2service/trips/left", info, &response)
	return response.Data, err
}

func (c *Travel2Client) RouteForTrip(ctx context.Context, tripID string) (*domain.Route, error) {
	var response domain.Response[*domain.Route]
	err := c.transport.exchange(ctx, travel2ServiceName, http.MethodGet, "/api/v1/travel2service/routes/"+url.PathEscape(tripID), nil, &response)
	return response.Data, err
}

func (c *Travel2Client) TripsByRoutes(ctx context.Context, routeIDs []string) ([][]domain.Trip, error) {
	var response domain.Response[[][]domain.Trip]
	err := c.transport.exchange(ctx, travel2ServiceName, http.MethodPost, "/api/v1/travel2service/trips/routes", routeIDs, &response)
	return response.Data, err
}

func (c *Travel2Client) TripDetail(ctx context.Context, info domain.TripAllDetailInfo) (*domain.TripAllDetail, error) {
	var response domain.Response[*domain.TripAllDetail]
	err := c.transport.exchange(ctx, travel2ServiceName, http.MethodPost, "/api/v1/travel2service/trip_detail", info, &response)
	return response.Data, err
}
