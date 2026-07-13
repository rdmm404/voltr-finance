package households

import (
	"context"
	"testing"
)

type fakeRepository struct{ name, guild string }

func (*fakeRepository) List(context.Context) ([]Household, error) { return nil, nil }
func (*fakeRepository) GetByID(_ context.Context, id int64) (Household, error) {
	return Household{ID: id}, nil
}
func (f *fakeRepository) GetByName(_ context.Context, value string) (Household, error) {
	f.name = value
	return Household{ID: 1}, nil
}
func (f *fakeRepository) GetByGuildID(_ context.Context, value string) (Household, error) {
	f.guild = value
	return Household{ID: 2}, nil
}
func (*fakeRepository) ListUsers(context.Context, int64) ([]User, error) { return nil, nil }

func TestServiceSupportsLookupResolutionAndUsers(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)
	if item, err := service.Get(context.Background(), 7); err != nil || item.ID != 7 {
		t.Fatalf("Get=%+v error=%v", item, err)
	}
	name := " Home "
	if _, err := service.Resolve(context.Background(), Selector{Name: &name}); err != nil || repo.name != "Home" {
		t.Fatalf("Resolve error=%v name=%q", err, repo.name)
	}
	guild := "guild"
	if _, err := service.Resolve(context.Background(), Selector{Name: &name, GuildID: &guild}); err == nil {
		t.Fatal("Resolve accepted two selectors")
	}
	items, err := service.List(context.Background())
	if err != nil || items == nil {
		t.Fatalf("List=%#v error=%v", items, err)
	}
	users, err := service.ListUsers(context.Background(), 7)
	if err != nil || users == nil {
		t.Fatalf("ListUsers=%#v error=%v", users, err)
	}
}
