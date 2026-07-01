package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/FudanSELab/train-ticket/ts-station-service/internal/domain"
	"github.com/FudanSELab/train-ticket/ts-station-service/internal/repository"
)

type Service struct{ repository repository.Repository }

func New(repository repository.Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(ctx context.Context, station domain.Station) (domain.Response, error) {
	if station.Name == nil {
		return domain.Response{}, errors.New("null station name")
	}
	name := normalize(*station.Name)
	station.Name = &name
	if name == "" {
		return domain.Response{Status: 0, Msg: "Name not specify", Data: station}, nil
	}
	existing, err := s.repository.FindByName(ctx, name)
	if err != nil {
		return domain.Response{}, err
	}
	if existing != nil {
		return domain.Response{Status: 0, Msg: "Already exists", Data: station}, nil
	}
	if station.ID == nil {
		id, err := uuid()
		if err != nil {
			return domain.Response{}, err
		}
		station.ID = &id
	}
	if err := s.repository.Insert(ctx, station); err != nil {
		return domain.Response{}, err
	}
	return domain.Response{Status: 1, Msg: "Create success", Data: station}, nil
}

func (s *Service) Update(ctx context.Context, info domain.Station) (domain.Response, error) {
	if info.ID == nil {
		return domain.Response{}, errors.New("null station id")
	}
	station, err := s.repository.FindByID(ctx, *info.ID)
	if err != nil {
		return domain.Response{}, err
	}
	if station == nil {
		return domain.Response{Status: 0, Msg: "Station not exist", Data: nil}, nil
	}
	if info.Name == nil {
		return domain.Response{}, errors.New("null station name")
	}
	name := normalize(*info.Name)
	station.Name, station.StayTime = &name, info.StayTime
	if err := s.repository.Update(ctx, *station); err != nil {
		return domain.Response{}, err
	}
	return domain.Response{Status: 1, Msg: "Update success", Data: *station}, nil
}

func (s *Service) Delete(ctx context.Context, id string) (domain.Response, error) {
	station, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return domain.Response{}, err
	}
	if station == nil {
		return domain.Response{Status: 0, Msg: "Station not exist", Data: nil}, nil
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return domain.Response{}, err
	}
	return domain.Response{Status: 1, Msg: "Delete success", Data: *station}, nil
}

func (s *Service) Query(ctx context.Context) (domain.Response, error) {
	stations, err := s.repository.FindAll(ctx)
	if err != nil {
		return domain.Response{}, err
	}
	if len(stations) == 0 {
		return domain.Response{Status: 0, Msg: "No content", Data: nil}, nil
	}
	return domain.Response{Status: 1, Msg: "Find all content", Data: stations}, nil
}

func (s *Service) QueryForID(ctx context.Context, name string) (domain.Response, error) {
	station, err := s.repository.FindByName(ctx, name)
	if err != nil {
		return domain.Response{}, err
	}
	if station == nil {
		return domain.Response{Status: 0, Msg: "Not exists", Data: name}, nil
	}
	return domain.Response{Status: 1, Msg: "Success", Data: value(station.ID)}, nil
}

func (s *Service) QueryForIDBatch(ctx context.Context, names []string) (domain.Response, error) {
	stations, err := s.repository.FindByNames(ctx, names)
	if err != nil {
		return domain.Response{}, err
	}
	known := make(map[string]string, len(stations))
	for _, station := range stations {
		known[value(station.Name)] = value(station.ID)
	}
	result := make(map[string]any, len(names))
	for _, name := range names {
		if id, ok := known[name]; ok {
			result[name] = id
		} else {
			result[name] = nil
		}
	}
	if len(result) == 0 {
		return domain.Response{Status: 0, Msg: "No content according to name list", Data: nil}, nil
	}
	return domain.Response{Status: 1, Msg: "Success", Data: result}, nil
}

func (s *Service) QueryByID(ctx context.Context, id string) (domain.Response, error) {
	station, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return domain.Response{}, err
	}
	if station == nil {
		return domain.Response{Status: 0, Msg: "No that stationId", Data: id}, nil
	}
	return domain.Response{Status: 1, Msg: "Success", Data: value(station.Name)}, nil
}

func (s *Service) QueryByIDBatch(ctx context.Context, ids []string) (domain.Response, error) {
	result := make([]string, 0)
	for _, id := range ids {
		station, err := s.repository.FindByID(ctx, id)
		if err != nil {
			return domain.Response{}, err
		}
		if station != nil {
			result = append(result, value(station.Name))
		}
	}
	if len(result) == 0 {
		return domain.Response{Status: 0, Msg: "No stationNamelist according to stationIdList", Data: result}, nil
	}
	return domain.Response{Status: 1, Msg: "Success", Data: result}, nil
}

func (s *Service) Seed(ctx context.Context) error {
	seeds := []struct {
		name string
		stay int
	}{
		{"Shang Hai", 10}, {"Shang Hai Hong Qiao", 10}, {"Tai Yuan", 5}, {"Bei Jing", 10},
		{"Nan Jing", 8}, {"Shi Jia Zhuang", 8}, {"Xu Zhou", 7}, {"Ji Nan", 5},
		{"Hang Zhou", 9}, {"Jia Xing Nan", 2}, {"Zhen Jiang", 2}, {"Wu Xi", 3}, {"Su Zhou", 3},
	}
	for _, seed := range seeds {
		name := seed.name
		if _, err := s.Create(ctx, domain.Station{Name: &name, StayTime: seed.stay}); err != nil {
			return err
		}
	}
	return nil
}

func normalize(value string) string { return strings.ToLower(strings.ReplaceAll(value, " ", "")) }
func value(pointer *string) string {
	if pointer == nil {
		return ""
	}
	return *pointer
}

func uuid() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16]), nil
}
