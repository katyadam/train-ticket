package testsupport

import (
	"context"
	"errors"
	"sync"

	"github.com/FudanSELab/train-ticket/ts-station-service/internal/domain"
)

type Repository struct {
	mu       sync.Mutex
	Stations map[string]domain.Station
}

func NewRepository() *Repository { return &Repository{Stations: make(map[string]domain.Station)} }

func (r *Repository) FindByName(_ context.Context, name string) (*domain.Station, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, station := range r.Stations {
		if value(station.Name) == name {
			copy := clone(station)
			return &copy, nil
		}
	}
	return nil, nil
}

func (r *Repository) FindByID(_ context.Context, id string) (*domain.Station, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	station, ok := r.Stations[id]
	if !ok {
		return nil, nil
	}
	copy := clone(station)
	return &copy, nil
}

func (r *Repository) FindByNames(_ context.Context, names []string) ([]domain.Station, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	wanted := make(map[string]bool, len(names))
	for _, name := range names {
		wanted[name] = true
	}
	result := make([]domain.Station, 0)
	for _, station := range r.Stations {
		if wanted[value(station.Name)] {
			result = append(result, clone(station))
		}
	}
	return result, nil
}

func (r *Repository) FindAll(_ context.Context) ([]domain.Station, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]domain.Station, 0, len(r.Stations))
	for _, station := range r.Stations {
		result = append(result, clone(station))
	}
	return result, nil
}

func (r *Repository) Insert(_ context.Context, station domain.Station) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, name := value(station.ID), value(station.Name)
	if _, found := r.Stations[id]; found {
		return errors.New("duplicate id")
	}
	for _, existing := range r.Stations {
		if value(existing.Name) == name {
			return errors.New("duplicate name")
		}
	}
	r.Stations[id] = clone(station)
	return nil
}

func (r *Repository) Update(_ context.Context, station domain.Station) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, name := value(station.ID), value(station.Name)
	if _, found := r.Stations[id]; !found {
		return errors.New("missing station")
	}
	for existingID, existing := range r.Stations {
		if existingID != id && value(existing.Name) == name {
			return errors.New("duplicate name")
		}
	}
	r.Stations[id] = clone(station)
	return nil
}

func (r *Repository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Stations, id)
	return nil
}

func clone(station domain.Station) domain.Station {
	id, name := value(station.ID), value(station.Name)
	station.ID, station.Name = &id, &name
	return station
}
func value(pointer *string) string {
	if pointer == nil {
		return ""
	}
	return *pointer
}
