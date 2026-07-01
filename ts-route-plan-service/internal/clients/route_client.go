package clients

import (
	"context"
	"net/http"
	"net/url"

	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/discovery"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/domain"
)

const routeServiceName = "ts-route-service"

type RouteClient struct{ transport transport }

func NewRouteClient(resolver discovery.Resolver, client *http.Client) *RouteClient {
	return &RouteClient{transport: newTransport(resolver, client)}
}

func (c *RouteClient) RoutesBetween(ctx context.Context, start, end string) ([]domain.Route, error) {
	var response domain.Response[[]domain.Route]
	path := "/api/v1/routeservice/routes/" + url.PathEscape(start) + "/" + url.PathEscape(end)
	err := c.transport.exchange(ctx, routeServiceName, http.MethodGet, path, nil, &response)
	return response.Data, err
}

func (c *RouteClient) RouteByID(ctx context.Context, routeID string) (*domain.Route, int, error) {
	var response domain.Response[*domain.Route]
	err := c.transport.exchange(ctx, routeServiceName, http.MethodGet, "/api/v1/routeservice/routes/"+url.PathEscape(routeID), nil, &response)
	return response.Data, response.Status, err
}
