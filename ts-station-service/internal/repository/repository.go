package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/FudanSELab/train-ticket/ts-station-service/internal/domain"
)

type Repository interface {
	FindByName(context.Context, string) (*domain.Station, error)
	FindByID(context.Context, string) (*domain.Station, error)
	FindByNames(context.Context, []string) ([]domain.Station, error)
	FindAll(context.Context) ([]domain.Station, error)
	Insert(context.Context, domain.Station) error
	Update(context.Context, domain.Station) error
	Delete(context.Context, string) error
}

type MySQL struct{ db *sql.DB }

func NewMySQL(db *sql.DB) *MySQL { return &MySQL{db: db} }

func (m *MySQL) EnsureSchema(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS station (
  id varchar(36) NOT NULL,
  name varchar(255) DEFAULT NULL,
  stay_time int NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY UK_gnneuc0peq2qi08yftdjhy7ok (name)
) ENGINE=InnoDB`)
	return err
}

func (m *MySQL) FindByName(ctx context.Context, name string) (*domain.Station, error) {
	return scanStation(m.db.QueryRowContext(ctx, "SELECT id, name, stay_time FROM station WHERE name = ?", name))
}

func (m *MySQL) FindByID(ctx context.Context, id string) (*domain.Station, error) {
	return scanStation(m.db.QueryRowContext(ctx, "SELECT id, name, stay_time FROM station WHERE id = ?", id))
}

func (m *MySQL) FindByNames(ctx context.Context, names []string) ([]domain.Station, error) {
	if len(names) == 0 {
		return []domain.Station{}, nil
	}
	arguments := make([]any, len(names))
	placeholders := make([]string, len(names))
	for i, name := range names {
		arguments[i], placeholders[i] = name, "?"
	}
	rows, err := m.db.QueryContext(ctx, "SELECT id, name, stay_time FROM station WHERE name IN ("+strings.Join(placeholders, ",")+")", arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStations(rows)
}

func (m *MySQL) FindAll(ctx context.Context) ([]domain.Station, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT id, name, stay_time FROM station")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStations(rows)
}

func (m *MySQL) Insert(ctx context.Context, station domain.Station) error {
	_, err := m.db.ExecContext(ctx, "INSERT INTO station (id, name, stay_time) VALUES (?, ?, ?)", value(station.ID), value(station.Name), station.StayTime)
	return err
}

func (m *MySQL) Update(ctx context.Context, station domain.Station) error {
	result, err := m.db.ExecContext(ctx, "UPDATE station SET name = ?, stay_time = ? WHERE id = ?", value(station.Name), station.StayTime, value(station.ID))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("station disappeared during update")
	}
	return nil
}

func (m *MySQL) Delete(ctx context.Context, id string) error {
	_, err := m.db.ExecContext(ctx, "DELETE FROM station WHERE id = ?", id)
	return err
}

type rowScanner interface{ Scan(...any) error }

func scanStation(row rowScanner) (*domain.Station, error) {
	var id, name sql.NullString
	var stayTime int
	if err := row.Scan(&id, &name, &stayTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &domain.Station{ID: nullableString(id), Name: nullableString(name), StayTime: stayTime}, nil
}

func scanStations(rows *sql.Rows) ([]domain.Station, error) {
	stations := make([]domain.Station, 0)
	for rows.Next() {
		station, err := scanStation(rows)
		if err != nil {
			return nil, err
		}
		if station == nil {
			return nil, fmt.Errorf("unexpected empty station row")
		}
		stations = append(stations, *station)
	}
	return stations, rows.Err()
}

func value(pointer *string) any {
	if pointer == nil {
		return nil
	}
	return *pointer
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
