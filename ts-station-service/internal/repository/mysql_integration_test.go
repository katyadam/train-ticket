package repository_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/FudanSELab/train-ticket/ts-station-service/internal/repository"
	"github.com/FudanSELab/train-ticket/ts-station-service/internal/service"
)

func TestMySQL57SchemaSeedAndRestart(t *testing.T) {
	dsn := os.Getenv("STATION_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set STATION_TEST_MYSQL_DSN to a disposable MySQL 5.7 database")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	stationRepository := repository.NewMySQL(db)
	if err := stationRepository.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	var tableName, ddl string
	if err := db.QueryRowContext(ctx, "SHOW CREATE TABLE station").Scan(&tableName, &ddl); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"`id` varchar(36) NOT NULL",
		"`name` varchar(255) DEFAULT NULL",
		"`stay_time` int(11) NOT NULL",
		"UNIQUE KEY `UK_gnneuc0peq2qi08yftdjhy7ok` (`name`)",
		"ENGINE=InnoDB",
	} {
		if !strings.Contains(ddl, fragment) {
			t.Fatalf("SHOW CREATE TABLE station missing %q:\n%s", fragment, ddl)
		}
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM station"); err != nil {
		t.Fatal(err)
	}
	stationService := service.New(stationRepository)
	if err := stationService.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	if err := stationService.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	stations, err := stationRepository.FindAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(stations) != 13 {
		t.Fatalf("station count after restart seed = %d", len(stations))
	}
}
