package users

import (
	"context"
	"testing"
)

type fakeRepository struct {
	created  CreateInput
	updated  Update
	telegram string
	list     []User
}

func (f *fakeRepository) Create(_ context.Context, input CreateInput) (User, error) {
	f.created = input
	return User{ID: 1, Name: input.Name}, nil
}
func (f *fakeRepository) Update(_ context.Context, id int64, update Update) (User, error) {
	f.updated = update
	return User{ID: id, Name: "user"}, nil
}
func (f *fakeRepository) GetByID(_ context.Context, id int64) (User, error) { return User{ID: id}, nil }
func (f *fakeRepository) GetByDiscordID(context.Context, string) (User, error) {
	return User{ID: 2}, nil
}
func (f *fakeRepository) GetByTelegramID(_ context.Context, id string) (User, error) {
	f.telegram = id
	return User{ID: 3}, nil
}
func (f *fakeRepository) GetByPhoneNumber(context.Context, string) (User, error) {
	return User{ID: 4}, nil
}
func (f *fakeRepository) GetByWhatsAppID(context.Context, string) (User, error) {
	return User{ID: 5}, nil
}
func (f *fakeRepository) List(context.Context) ([]User, error) { return f.list, nil }

func TestServicePreservesSelectorsAndUpdates(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)
	if _, err := service.Create(context.Background(), CreateInput{Name: "  Rafael  "}); err != nil || repo.created.Name != "Rafael" {
		t.Fatalf("Create error=%v input=%+v", err, repo.created)
	}
	if _, err := service.Resolve(context.Background(), Selector{TelegramID: stringPointer("123|name")}); err != nil || repo.telegram != "123" {
		t.Fatalf("Resolve error=%v telegram=%q", err, repo.telegram)
	}
	if _, err := service.Resolve(context.Background(), Selector{DiscordID: stringPointer("d"), PhoneNumber: stringPointer("p")}); err == nil {
		t.Fatal("Resolve accepted multiple selectors")
	}
	if _, err := service.Update(context.Background(), UpdateInput{ID: 1, ClearPhoneNumber: true}); err != nil || !repo.updated.SetPhoneNumber || repo.updated.PhoneNumber != nil {
		t.Fatalf("Update error=%v update=%+v", err, repo.updated)
	}
	items, err := service.List(context.Background())
	if err != nil || items == nil {
		t.Fatalf("List=%#v error=%v", items, err)
	}
}

func stringPointer(value string) *string { return &value }
