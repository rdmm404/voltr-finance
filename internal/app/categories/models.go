package categories

import "rdmm404/voltr-finance/internal/app/patch"

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
