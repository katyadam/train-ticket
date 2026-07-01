package domain

// Response is the response envelope shared by Train Ticket services.
type Response[T any] struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   T      `json:"data"`
}

type RoutePlanInfo struct {
	StartStation *string `json:"startStation"`
	EndStation   *string `json:"endStation"`
	TravelDate   *string `json:"travelDate"`
	Num          int     `json:"num"`
}

type TripInfo struct {
	StartPlace    *string `json:"startPlace"`
	EndPlace      *string `json:"endPlace"`
	DepartureTime *string `json:"departureTime"`
}

type TripID struct {
	Type   *string `json:"type"`
	Number *string `json:"number"`
}

func (id TripID) String() (string, bool) {
	if id.Type == nil || id.Number == nil {
		return "", false
	}
	return *id.Type + *id.Number, true
}

type TripResponse struct {
	TripID               *TripID `json:"tripId"`
	TrainTypeName        *string `json:"trainTypeName"`
	StartStation         *string `json:"startStation"`
	TerminalStation      *string `json:"terminalStation"`
	StartTime            *string `json:"startTime"`
	EndTime              *string `json:"endTime"`
	EconomyClass         int     `json:"economyClass"`
	ConfortClass         int     `json:"confortClass"`
	PriceForEconomyClass *string `json:"priceForEconomyClass"`
	PriceForConfortClass *string `json:"priceForConfortClass"`
}

type RoutePlanResultUnit struct {
	TripID                  *string   `json:"tripId"`
	TrainTypeName           *string   `json:"trainTypeName"`
	StartStation            *string   `json:"startStation"`
	EndStation              *string   `json:"endStation"`
	StopStations            *[]string `json:"stopStations"`
	PriceForSecondClassSeat *string   `json:"priceForSecondClassSeat"`
	PriceForFirstClassSeat  *string   `json:"priceForFirstClassSeat"`
	StartTime               *string   `json:"startTime"`
	EndTime                 *string   `json:"endTime"`
}

type Route struct {
	ID           *string   `json:"id"`
	Stations     *[]string `json:"stations"`
	Distances    *[]int    `json:"distances"`
	StartStation *string   `json:"startStation"`
	EndStation   *string   `json:"endStation"`
}

type Trip struct {
	ID                  *string `json:"id"`
	TripID              *TripID `json:"tripId"`
	TrainTypeName       *string `json:"trainTypeName"`
	RouteID             *string `json:"routeId"`
	StartStationName    *string `json:"startStationName"`
	StationsName        *string `json:"stationsName"`
	TerminalStationName *string `json:"terminalStationName"`
	StartTime           *string `json:"startTime"`
	EndTime             *string `json:"endTime"`
}

type TripAllDetail struct {
	Status       bool          `json:"status"`
	Message      *string       `json:"message"`
	TripResponse *TripResponse `json:"tripResponse"`
	Trip         *Trip         `json:"trip"`
}

type TripAllDetailInfo struct {
	TripID     string  `json:"tripId"`
	TravelDate *string `json:"travelDate"`
	From       *string `json:"from"`
	To         *string `json:"to"`
}
