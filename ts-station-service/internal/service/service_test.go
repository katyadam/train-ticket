package service

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/FudanSELab/train-ticket/ts-station-service/internal/domain"
	"github.com/FudanSELab/train-ticket/ts-station-service/internal/testsupport"
)

func TestSeedIsIdempotentAndPreservesThirteenStations(t *testing.T) {
	repository := testsupport.NewRepository()
	service := New(repository)
	if err := service.Seed(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := service.Seed(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repository.Stations) != 13 {
		t.Fatalf("seed count = %d", len(repository.Stations))
	}
	got := make([]string, 0, len(repository.Stations))
	for _, station := range repository.Stations {
		got = append(got, deref(station.Name))
	}
	sort.Strings(got)
	want := []string{"beijing", "hangzhou", "jinan", "jiaxingnan", "nanjing", "shanghai", "shanghaihongqiao", "shijiazhuang", "suzhou", "taiyuan", "wuxi", "xuzhou", "zhenjiang"}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stations = %v", got)
	}
}

func TestCreateNormalizesAndPreservesDomainFailures(t *testing.T) {
	repository := testsupport.NewRepository()
	service := New(repository)
	name := " Shang Hai "
	created, err := service.Create(context.Background(), domain.Station{Name: &name, StayTime: 10})
	if err != nil {
		t.Fatal(err)
	}
	station := created.Data.(domain.Station)
	if created.Status != 1 || created.Msg != "Create success" || deref(station.Name) != "shanghai" || len(deref(station.ID)) != 36 {
		t.Fatalf("created = %#v", created)
	}

	duplicateName, suppliedID := "shanghai", "caller-id"
	duplicate, err := service.Create(context.Background(), domain.Station{ID: &suppliedID, Name: &duplicateName})
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.Status != 0 || duplicate.Msg != "Already exists" {
		t.Fatalf("duplicate = %#v", duplicate)
	}

	empty := "   "
	missing, err := service.Create(context.Background(), domain.Station{Name: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if missing.Status != 0 || missing.Msg != "Name not specify" {
		t.Fatalf("empty = %#v", missing)
	}

	newName := "Nanjing"
	withID, err := service.Create(context.Background(), domain.Station{ID: &suppliedID, Name: &newName})
	if err != nil {
		t.Fatal(err)
	}
	if deref(withID.Data.(domain.Station).ID) != suppliedID {
		t.Fatalf("caller ID not preserved: %#v", withID)
	}
}

func TestUpdateDeleteAndQueries(t *testing.T) {
	repository := testsupport.NewRepository()
	service := New(repository)
	ctx := context.Background()
	id, name := "id-1", "old"
	if err := repository.Insert(ctx, domain.Station{ID: &id, Name: &name, StayTime: 1}); err != nil {
		t.Fatal(err)
	}
	newName := " New Name "
	updated, err := service.Update(ctx, domain.Station{ID: &id, Name: &newName, StayTime: 9})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Msg != "Update success" || deref(updated.Data.(domain.Station).Name) != "newname" {
		t.Fatalf("updated = %#v", updated)
	}

	missingID, anyName := "missing", "x"
	missing, err := service.Update(ctx, domain.Station{ID: &missingID, Name: &anyName})
	if err != nil || missing.Status != 0 || missing.Data != nil {
		t.Fatalf("missing update = %#v err=%v", missing, err)
	}

	deleted, err := service.Delete(ctx, id)
	if err != nil || deleted.Msg != "Delete success" {
		t.Fatalf("deleted = %#v err=%v", deleted, err)
	}
	deletedAgain, err := service.Delete(ctx, id)
	if err != nil || deletedAgain.Msg != "Station not exist" {
		t.Fatalf("missing delete = %#v err=%v", deletedAgain, err)
	}
}

func TestBatchLookupSemanticsAndOrder(t *testing.T) {
	repository := testsupport.NewRepository()
	service := New(repository)
	ctx := context.Background()
	idA, idB, nameA, nameB := "a", "b", "alpha", "beta"
	_ = repository.Insert(ctx, domain.Station{ID: &idA, Name: &nameA})
	_ = repository.Insert(ctx, domain.Station{ID: &idB, Name: &nameB})

	ids, err := service.QueryForIDBatch(ctx, []string{"alpha", "unknown", "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	gotMap := ids.Data.(map[string]any)
	if gotMap["alpha"] != "a" || gotMap["unknown"] != nil {
		t.Fatalf("ID map = %#v", gotMap)
	}

	names, err := service.QueryByIDBatch(ctx, []string{"b", "missing", "a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names.Data, []string{"beta", "alpha", "beta"}) {
		t.Fatalf("name list = %#v", names.Data)
	}

	empty, err := service.QueryByIDBatch(ctx, []string{"missing"})
	if err != nil || empty.Status != 0 || !reflect.DeepEqual(empty.Data, []string{}) {
		t.Fatalf("empty = %#v err=%v", empty, err)
	}
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
