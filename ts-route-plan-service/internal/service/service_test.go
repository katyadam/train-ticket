package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/clients"
	"github.com/FudanSELab/train-ticket/ts-route-plan-service/internal/domain"
)

type recordedCall struct {
	target string
	method string
	path   string
	body   string
}

type recorder struct {
	mu    sync.Mutex
	calls []recordedCall
}

func (r *recorder) add(call recordedCall) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, call)
}

type staticResolver map[string]string

func (r staticResolver) Resolve(_ context.Context, name string) (string, error) { return r[name], nil }

func TestCheapestPreservesSelectionAndCallOrder(t *testing.T) {
	record := &recorder{}
	travel := stub(t, "ts-travel-service", record, func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/api/v1/travelservice/trips/left":
			writeJSON(t, w, envelope([]domain.TripResponse{
				tripResponse("G2", "10.0", "2020-01-01 10:00:00", "2020-01-01 12:00:00"),
				tripResponse("G1", "5.0", "2020-01-01 10:00:00", "2020-01-01 11:00:00"),
			}))
		case "/api/v1/travelservice/routes/G1", "/api/v1/travelservice/routes/G2":
			writeJSON(t, w, envelope(&domain.Route{Stations: stringSlice("shanghai", "beijing")}))
		default:
			http.NotFound(w, req)
		}
	})
	defer travel.Close()
	travel2 := stub(t, "ts-travel2-service", record, func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/api/v1/travel2service/trips/left":
			writeJSON(t, w, envelope([]domain.TripResponse{
				tripResponse("K1", "5.0", "2020-01-01 10:00:00", "2020-01-01 13:00:00"),
				tripResponse("Z1", "20.0", "2020-01-01 10:00:00", "2020-01-01 14:00:00"),
			}))
		case "/api/v1/travel2service/routes/K1", "/api/v1/travel2service/routes/Z1":
			writeJSON(t, w, envelope(&domain.Route{Stations: stringSlice("shanghai", "beijing")}))
		default:
			http.NotFound(w, req)
		}
	})
	defer travel2.Close()
	route := stub(t, "ts-route-service", record, http.NotFound)
	defer route.Close()

	svc := newTestService(travel.URL, travel2.URL, route.URL)
	start, end, date := " Shang Hai ", "Bei Jing", "2020-01-01"
	response, err := svc.Cheapest(context.Background(), domain.RoutePlanInfo{StartStation: &start, EndStation: &end, TravelDate: &date, Num: 1})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != 1 || response.Msg != "Success" {
		t.Fatalf("unexpected response: %#v", response)
	}
	wantTrips := []string{"G1", "K1", "G2", "Z1"}
	gotTrips := make([]string, 0, len(response.Data))
	for _, unit := range response.Data {
		gotTrips = append(gotTrips, deref(unit.TripID))
	}
	if !reflect.DeepEqual(gotTrips, wantTrips) {
		t.Fatalf("trip order: got %v want %v", gotTrips, wantTrips)
	}

	wantCalls := []recordedCall{
		{"ts-travel-service", "POST", "/api/v1/travelservice/trips/left", `{"startPlace":"shanghai","endPlace":"beijing","departureTime":"2020-01-01"}`},
		{"ts-travel2-service", "POST", "/api/v1/travel2service/trips/left", `{"startPlace":"shanghai","endPlace":"beijing","departureTime":"2020-01-01"}`},
		{"ts-travel-service", "GET", "/api/v1/travelservice/routes/G1", ""},
		{"ts-travel2-service", "GET", "/api/v1/travel2service/routes/K1", ""},
		{"ts-travel-service", "GET", "/api/v1/travelservice/routes/G2", ""},
		{"ts-travel2-service", "GET", "/api/v1/travel2service/routes/Z1", ""},
	}
	if !reflect.DeepEqual(record.calls, wantCalls) {
		t.Fatalf("calls:\n got %#v\nwant %#v", record.calls, wantCalls)
	}
}

func TestQuickestPreservesTieOrderAndEpochFallback(t *testing.T) {
	record := &recorder{}
	travel := stub(t, "ts-travel-service", record, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/api/v1/travelservice/trips/left" {
			writeJSON(t, w, envelope([]domain.TripResponse{
				tripResponse("G1", "9", "bad", "bad"),
				tripResponse("D1", "8", "2020-01-01 10:00:00", "2020-01-01 11:00:00"),
			}))
			return
		}
		writeJSON(t, w, envelope(&domain.Route{Stations: stringSlice("a", "b")}))
	})
	defer travel.Close()
	travel2 := stub(t, "ts-travel2-service", record, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/api/v1/travel2service/trips/left" {
			writeJSON(t, w, envelope([]domain.TripResponse{tripResponse("K1", "7", "2020-01-01 10:00:00", "2020-01-01 11:00:00")}))
			return
		}
		writeJSON(t, w, envelope(&domain.Route{Stations: stringSlice("a", "b")}))
	})
	defer travel2.Close()
	route := stub(t, "ts-route-service", record, http.NotFound)
	defer route.Close()

	a, b := "a", "b"
	response, err := newTestService(travel.URL, travel2.URL, route.URL).Quickest(context.Background(), domain.RoutePlanInfo{StartStation: &a, EndStation: &b})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"G1", "D1", "K1"}
	var got []string
	for _, unit := range response.Data {
		got = append(got, deref(unit.TripID))
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("trip order: got %v want %v", got, want)
	}
}

func TestMinimumStopsPreservesMergeDirectionAndEndpointDispatch(t *testing.T) {
	record := &recorder{}
	travel := stub(t, "ts-travel-service", record, detailStub(t, "travel", "G", false))
	defer travel.Close()
	travel2 := stub(t, "ts-travel2-service", record, detailStub(t, "travel2", "K", false))
	defer travel2.Close()
	route := stub(t, "ts-route-service", record, func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/api/v1/routeservice/routes/s/e":
			r1, r2 := "r1", "r2"
			writeJSON(t, w, envelope([]domain.Route{
				{ID: &r1, Stations: stringSlice("s", "x", "e")},
				{ID: &r2, Stations: stringSlice("s", "e")},
			}))
		case "/api/v1/routeservice/routes/route-K2":
			writeJSON(t, w, domain.Response[*domain.Route]{Status: 0, Msg: "not found", Data: nil})
		default:
			writeJSON(t, w, envelope(&domain.Route{Stations: stringSlice("s", "e")}))
		}
	})
	defer route.Close()

	s, e, date := "s", "e", "2020-01-01"
	response, err := newTestService(travel.URL, travel2.URL, route.URL).MinStops(context.Background(), domain.RoutePlanInfo{StartStation: &s, EndStation: &e, TravelDate: &date})
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg != "Success." {
		t.Fatalf("message = %q", response.Msg)
	}
	wantTrips := []string{"K2", "G2", "K1", "G1"}
	var gotTrips []string
	for _, unit := range response.Data {
		gotTrips = append(gotTrips, deref(unit.TripID))
	}
	if !reflect.DeepEqual(gotTrips, wantTrips) {
		t.Fatalf("merge order: got %v want %v", gotTrips, wantTrips)
	}
	if response.Data[0].StopStations != nil {
		t.Fatalf("status-0 route should leave stopStations null: %#v", response.Data[0])
	}

	wantPrefix := []recordedCall{
		{"ts-route-service", "GET", "/api/v1/routeservice/routes/s/e", ""},
		{"ts-travel-service", "POST", "/api/v1/travelservice/trips/routes", `["r2","r1"]`},
		{"ts-travel2-service", "POST", "/api/v1/travel2service/trips/routes", `["r2","r1"]`},
		{"ts-travel2-service", "POST", "/api/v1/travel2service/trip_detail", `{"tripId":"K2","travelDate":"2020-01-01","from":"s","to":"e"}`},
		{"ts-route-service", "GET", "/api/v1/routeservice/routes/route-K2", ""},
	}
	if len(record.calls) < len(wantPrefix) || !reflect.DeepEqual(record.calls[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("call prefix:\n got %#v\nwant %#v", record.calls, wantPrefix)
	}
}

func detailStub(t *testing.T, serviceName, prefix string, _ bool) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.URL.Path == "/api/v1/"+serviceName+"service/trips/routes":
			one, two := prefix+"2", prefix+"1"
			routeOne, routeTwo := "route-"+one, "route-"+two
			writeJSON(t, w, envelope([][]domain.Trip{
				{{TripID: tripID(one), RouteID: &routeOne}},
				{{TripID: tripID(two), RouteID: &routeTwo}},
			}))
		case req.URL.Path == "/api/v1/"+serviceName+"service/trip_detail":
			var input domain.TripAllDetailInfo
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				t.Fatal(err)
			}
			response := tripResponse(input.TripID, "1", "2020-01-01 10:00:00", "2020-01-01 11:00:00")
			writeJSON(t, w, envelope(&domain.TripAllDetail{Status: true, TripResponse: &response}))
		default:
			http.NotFound(w, req)
		}
	}
}

func newTestService(travelURL, travel2URL, routeURL string) *Service {
	resolver := staticResolver{
		"ts-travel-service": travelURL, "ts-travel2-service": travel2URL, "ts-route-service": routeURL,
	}
	client := &http.Client{}
	return New(clients.NewTravelClient(resolver, client), clients.NewTravel2Client(resolver, client), clients.NewRouteClient(resolver, client))
}

func stub(t *testing.T, target string, record *recorder, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		req.Body.Close()
		req.Body = io.NopCloser(stringsReader(string(body)))
		record.add(recordedCall{target: target, method: req.Method, path: req.URL.Path, body: string(body)})
		handler(w, req)
	}))
}

type stringReader string

func (r stringReader) Read(p []byte) (int, error) {
	if len(r) == 0 {
		return 0, io.EOF
	}
	n := copy(p, string(r))
	return n, io.EOF
}

func stringsReader(value string) io.Reader { return &reader{value: value} }

type reader struct{ value string }

func (r *reader) Read(p []byte) (int, error) {
	if r.value == "" {
		return 0, io.EOF
	}
	n := copy(p, r.value)
	r.value = r.value[n:]
	return n, nil
}

func envelope[T any](data T) domain.Response[T] {
	return domain.Response[T]{Status: 1, Msg: "Success", Data: data}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}

func tripResponse(id, price, start, end string) domain.TripResponse {
	trainType, startStation, terminal, comfort := "train", "s", "e", "2"
	return domain.TripResponse{
		TripID: tripID(id), TrainTypeName: &trainType, StartStation: &startStation, TerminalStation: &terminal,
		StartTime: &start, EndTime: &end, PriceForEconomyClass: &price, PriceForConfortClass: &comfort,
	}
}

func tripID(value string) *domain.TripID {
	typeValue, number := value[:1], value[1:]
	return &domain.TripID{Type: &typeValue, Number: &number}
}

func stringSlice(values ...string) *[]string { return &values }

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
