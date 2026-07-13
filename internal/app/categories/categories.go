package categories

import (
	"context"
	"regexp"
	"strings"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
	"rdmm404/voltr-finance/internal/app/patch"
)

var codePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Category struct {
	ID          int64
	Code        string
	Name        string
	Description *string
	IsActive    bool
}

type CreateInput struct {
	Name        string
	Code        *string
	Description *string
}

type UpdateInput struct {
	Code        string
	Name        *string
	Description patch.Field[string]
}

type Update struct {
	Name        *string
	Description patch.Field[string]
}

// Repository implementations translate missing rows and unique violations to
// apperrors.KindNotFound and apperrors.KindConflict respectively.
type Repository interface {
	Create(context.Context, CreateInput) (Category, error)
	List(context.Context, bool) ([]Category, error)
	GetByCode(context.Context, string) (Category, error)
	GetActiveByID(context.Context, int64) (Category, error)
	GetActiveByCode(context.Context, string) (Category, error)
	Update(context.Context, string, Update) (Category, error)
	Deactivate(context.Context, string) (Category, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, input CreateInput) (Category, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Category{}, apperrors.Validation("category name is required")
	}
	code := slug(input.Name)
	if input.Code != nil {
		code = strings.TrimSpace(*input.Code)
	}
	if !codePattern.MatchString(code) {
		return Category{}, apperrors.Validation("category code must be a lowercase slug")
	}
	input.Code = &code
	item, err := s.repo.Create(ctx, input)
	return item, apperrors.WrapInternal("create category", err)
}

func (s *Service) List(ctx context.Context, includeInactive bool) ([]Category, error) {
	items, err := s.repo.List(ctx, includeInactive)
	if items == nil && err == nil {
		items = []Category{}
	}
	return items, apperrors.WrapInternal("list categories", err)
}

func (s *Service) GetByCode(ctx context.Context, code string) (Category, error) {
	code, err := validateCode(code)
	if err != nil {
		return Category{}, err
	}
	item, repoErr := s.repo.GetByCode(ctx, code)
	return item, apperrors.WrapInternal("get category", repoErr)
}

func (s *Service) ResolveActive(ctx context.Context, id *int64, code *string) (Category, error) {
	if id == nil && code == nil {
		return Category{}, apperrors.Validation("category selector is required")
	}
	if id != nil && code != nil {
		byID, err := s.repo.GetActiveByID(ctx, *id)
		if err != nil {
			return Category{}, apperrors.WrapInternal("resolve category", err)
		}
		byCode, err := s.repo.GetActiveByCode(ctx, strings.TrimSpace(*code))
		if err != nil {
			return Category{}, apperrors.WrapInternal("resolve category", err)
		}
		if byID.ID != byCode.ID {
			return Category{}, apperrors.Validation("category id and code refer to different categories")
		}
		return byID, nil
	}
	if id != nil {
		item, err := s.repo.GetActiveByID(ctx, *id)
		return item, apperrors.WrapInternal("resolve category", err)
	}
	item, err := s.repo.GetActiveByCode(ctx, strings.TrimSpace(*code))
	return item, apperrors.WrapInternal("resolve category", err)
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (Category, error) {
	code, err := validateCode(input.Code)
	if err != nil {
		return Category{}, err
	}
	if input.Name == nil && !input.Description.Present() {
		return Category{}, apperrors.Validation("at least one category field is required")
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return Category{}, apperrors.Validation("category name is required")
		}
		input.Name = &name
	}
	item, err := s.repo.Update(ctx, code, Update{Name: input.Name, Description: input.Description})
	return item, apperrors.WrapInternal("update category", err)
}

func (s *Service) Deactivate(ctx context.Context, code string) (Category, error) {
	code, err := validateCode(code)
	if err != nil {
		return Category{}, err
	}
	item, repoErr := s.repo.Deactivate(ctx, code)
	return item, apperrors.WrapInternal("deactivate category", repoErr)
}

func validateCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	if !codePattern.MatchString(code) {
		return "", apperrors.Validation("category code must be a lowercase slug")
	}
	return code, nil
}

func slug(name string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
