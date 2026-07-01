package service

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/clients"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/domain"
)

type Service struct {
	travel   *clients.TravelClient
	travel2  *clients.Travel2Client
	route    *clients.RouteClient
	location *time.Location
}

func New(travel *clients.TravelClient, travel2 *clients.Travel2Client, route *clients.RouteClient) *Service {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return &Service{travel: travel, travel2: travel2, route: route, location: location}
}

func (s *Service) Cheapest(ctx context.Context, info domain.RoutePlanInfo) (domain.Response[[]domain.RoutePlanResultUnit], error) {
	query := tripQuery(info)
	highSpeed, err := s.travel.LeftTrips(ctx, query)
	if err != nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, err
	}
	normal, err := s.travel2.LeftTrips(ctx, query)
	if err != nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, err
	}
	all := append(append([]domain.TripResponse{}, highSpeed...), normal...)
	selected, err := selectByPrice(all)
	if err != nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, err
	}
	units, err := s.unitsWithStations(ctx, selected)
	if err != nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, err
	}
	return domain.Response[[]domain.RoutePlanResultUnit]{Status: 1, Msg: "Success", Data: units}, nil
}

func (s *Service) Quickest(ctx context.Context, info domain.RoutePlanInfo) (domain.Response[[]domain.RoutePlanResultUnit], error) {
	query := tripQuery(info)
	highSpeed, err := s.travel.LeftTrips(ctx, query)
	if err != nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, err
	}
	normal, err := s.travel2.LeftTrips(ctx, query)
	if err != nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, err
	}
	all := append(append([]domain.TripResponse{}, highSpeed...), normal...)
	selected, err := s.selectByDuration(all)
	if err != nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, err
	}
	units, err := s.unitsWithStations(ctx, selected)
	if err != nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, err
	}
	return domain.Response[[]domain.RoutePlanResultUnit]{Status: 1, Msg: "Success", Data: units}, nil
}

func (s *Service) MinStops(ctx context.Context, info domain.RoutePlanInfo) (domain.Response[[]domain.RoutePlanResultUnit], error) {
	start, end := normalized(info.StartStation), normalized(info.EndStation)
	if start == nil || end == nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, errors.New("null station")
	}
	routes, err := s.route.RoutesBetween(ctx, *start, *end)
	if err != nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, err
	}
	selectedRouteIDs := selectRoutes(routes, *start, *end)
	travelTrips, err := s.travel.TripsByRoutes(ctx, selectedRouteIDs)
	if err != nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, err
	}
	travel2Trips, err := s.travel2.TripsByRoutes(ctx, selectedRouteIDs)
	if err != nil {
		return domain.Response[[]domain.RoutePlanResultUnit]{}, err
	}

	trips := make([]domain.Trip, 0)
	for i := range travel2Trips {
		if i >= len(travelTrips) {
			return domain.Response[[]domain.RoutePlanResultUnit]{}, errors.New("travel result list length mismatch")
		}
		merged := append(append([]domain.Trip{}, travel2Trips[i]...), travelTrips[i]...)
		trips = append(trips, merged...)
	}

	units := make([]domain.RoutePlanResultUnit, 0, len(trips))
	for _, trip := range trips {
		if trip.TripID == nil {
			return domain.Response[[]domain.RoutePlanResultUnit]{}, errors.New("null tripId")
		}
		tripID, ok := trip.TripID.String()
		if !ok || tripID == "" {
			return domain.Response[[]domain.RoutePlanResultUnit]{}, errors.New("invalid tripId")
		}
		detailInfo := domain.TripAllDetailInfo{TripID: tripID, TravelDate: info.TravelDate, From: start, To: end}
		var detail *domain.TripAllDetail
		if tripID[0] == 'D' || tripID[0] == 'G' {
			detail, err = s.travel.TripDetail(ctx, detailInfo)
		} else {
			detail, err = s.travel2.TripDetail(ctx, detailInfo)
		}
		if err != nil {
			return domain.Response[[]domain.RoutePlanResultUnit]{}, err
		}
		if detail == nil || detail.TripResponse == nil {
			return domain.Response[[]domain.RoutePlanResultUnit]{}, errors.New("null trip detail")
		}
		response := detail.TripResponse
		unit := domain.RoutePlanResultUnit{
			TripID: &tripID, TrainTypeName: response.TrainTypeName,
			StartStation: response.StartStation, EndStation: response.TerminalStation,
			PriceForSecondClassSeat: response.PriceForEconomyClass,
			PriceForFirstClassSeat:  response.PriceForConfortClass,
			StartTime:               response.StartTime, EndTime: response.EndTime,
		}
		if trip.RouteID == nil {
			return domain.Response[[]domain.RoutePlanResultUnit]{}, errors.New("null routeId")
		}
		route, status, routeErr := s.route.RouteByID(ctx, *trip.RouteID)
		if routeErr != nil {
			return domain.Response[[]domain.RoutePlanResultUnit]{}, routeErr
		}
		if status != 0 {
			if route == nil {
				return domain.Response[[]domain.RoutePlanResultUnit]{}, errors.New("null route")
			}
			unit.StopStations = route.Stations
		}
		units = append(units, unit)
	}
	return domain.Response[[]domain.RoutePlanResultUnit]{Status: 1, Msg: "Success.", Data: units}, nil
}

func tripQuery(info domain.RoutePlanInfo) domain.TripInfo {
	return domain.TripInfo{StartPlace: normalized(info.StartStation), EndPlace: normalized(info.EndStation), DepartureTime: info.TravelDate}
}

func normalized(value *string) *string {
	if value == nil || *value == "" {
		return value
	}
	normalized := strings.ToLower(strings.ReplaceAll(*value, " ", ""))
	return &normalized
}

func selectByPrice(input []domain.TripResponse) ([]domain.TripResponse, error) {
	remaining := append([]domain.TripResponse{}, input...)
	selected := make([]domain.TripResponse, 0, min(5, len(remaining)))
	minIndex := -1
	for i := 0; i < min(5, len(input)); i++ {
		minPrice := float32(math.MaxFloat32)
		for j, result := range remaining {
			if result.PriceForEconomyClass == nil {
				return nil, errors.New("null economy price")
			}
			price64, err := strconv.ParseFloat(*result.PriceForEconomyClass, 32)
			if err != nil {
				return nil, err
			}
			price := float32(price64)
			if price < minPrice {
				minPrice, minIndex = price, j
			}
		}
		if minIndex < 0 || minIndex >= len(remaining) {
			return nil, errors.New("no minimum price")
		}
		selected = append(selected, remaining[minIndex])
		remaining = append(remaining[:minIndex], remaining[minIndex+1:]...)
	}
	return selected, nil
}

func (s *Service) selectByDuration(input []domain.TripResponse) ([]domain.TripResponse, error) {
	remaining := append([]domain.TripResponse{}, input...)
	selected := make([]domain.TripResponse, 0, min(5, len(remaining)))
	minIndex := -1
	for i := 0; i < min(5, len(input)); i++ {
		minTime := int64(math.MaxInt64)
		for j, result := range remaining {
			end := s.javaDate(result.EndTime)
			start := s.javaDate(result.StartTime)
			duration := end.UnixMilli() - start.UnixMilli()
			if duration < minTime {
				minTime, minIndex = duration, j
			}
		}
		if minIndex < 0 || minIndex >= len(remaining) {
			return nil, errors.New("no minimum duration")
		}
		selected = append(selected, remaining[minIndex])
		remaining = append(remaining[:minIndex], remaining[minIndex+1:]...)
	}
	return selected, nil
}

func (s *Service) unitsWithStations(ctx context.Context, trips []domain.TripResponse) ([]domain.RoutePlanResultUnit, error) {
	units := make([]domain.RoutePlanResultUnit, 0, len(trips))
	for _, response := range trips {
		if response.TripID == nil {
			return nil, errors.New("null tripId")
		}
		tripID, ok := response.TripID.String()
		if !ok || tripID == "" {
			return nil, errors.New("invalid tripId")
		}
		var route *domain.Route
		var err error
		if tripID[0] == 'G' || tripID[0] == 'D' {
			route, err = s.travel.RouteForTrip(ctx, tripID)
		} else {
			route, err = s.travel2.RouteForTrip(ctx, tripID)
		}
		if err != nil {
			return nil, err
		}
		if route == nil {
			return nil, errors.New("null route")
		}
		units = append(units, domain.RoutePlanResultUnit{
			TripID: &tripID, TrainTypeName: response.TrainTypeName,
			StartStation: response.StartStation, EndStation: response.TerminalStation,
			StopStations:            route.Stations,
			PriceForSecondClassSeat: response.PriceForEconomyClass,
			PriceForFirstClassSeat:  response.PriceForConfortClass,
			StartTime:               response.StartTime, EndTime: response.EndTime,
		})
	}
	return units, nil
}

func selectRoutes(routes []domain.Route, start, end string) []string {
	remaining := append([]domain.Route{}, routes...)
	gaps := make([]int, len(remaining))
	for i, route := range remaining {
		if route.Stations == nil {
			gaps[i] = 0
			continue
		}
		gaps[i] = javaIndex(*route.Stations, end) - javaIndex(*route.Stations, start)
	}
	result := make([]string, 0, min(5, len(remaining)))
	for i := 0; i < min(5, len(routes)); i++ {
		minIndex, minGap := 0, int(^uint(0)>>1)
		for j, gap := range gaps {
			if gap < minGap {
				minGap, minIndex = gap, j
			}
		}
		if remaining[minIndex].ID == nil {
			return result
		}
		result = append(result, *remaining[minIndex].ID)
		remaining = append(remaining[:minIndex], remaining[minIndex+1:]...)
		gaps = append(gaps[:minIndex], gaps[minIndex+1:]...)
	}
	return result
}

func javaIndex(values []string, wanted string) int {
	for i, value := range values {
		if value == wanted {
			return i
		}
	}
	return -1
}

var dateOnly = regexp.MustCompile(`^([0-9]{1,4})-([0-9]{1,2})-([0-9]{1,2})`)
var dateTime = regexp.MustCompile(`^([0-9]{1,4})-([0-9]{1,2})-([0-9]{1,2}) ([0-9]{1,2}):([0-9]{1,2}):([0-9]{1,2})`)

func (s *Service) javaDate(value *string) time.Time {
	if value == nil {
		return time.Unix(0, 0)
	}
	matcher := dateOnly
	if len(*value) > 10 {
		matcher = dateTime
	}
	parts := matcher.FindStringSubmatch(*value)
	if parts == nil {
		return time.Unix(0, 0)
	}
	numbers := make([]int, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		number, err := strconv.Atoi(parts[i])
		if err != nil {
			return time.Unix(0, 0)
		}
		numbers[i-1] = number
	}
	hour, minute, second := 0, 0, 0
	if len(numbers) == 6 {
		hour, minute, second = numbers[3], numbers[4], numbers[5]
	}
	return time.Date(numbers[0], time.Month(numbers[1]), numbers[2], hour, minute, second, 0, s.location)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
