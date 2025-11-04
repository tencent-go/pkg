package page

import (
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/types"
	"github.com/tencent-go/pkg/validation"
)

type Pagination struct {
	Current  int64 `json:"current,omitempty" query:"current,omitempty" validate:"omitempty"`   // 当前页码
	PageSize int64 `json:"pageSize,omitempty" query:"pageSize,omitempty" validate:"omitempty"` // 每页数量
}

// Deprecated: use types.CursorQuery instead
type CursorPagination struct {
	Limit       int64     `json:"limit" query:"limit" validate:"omitempty"`
	StartCursor *types.ID `json:"startCursor,omitempty" query:"startCursor,omitempty" validate:"omitempty"`
}

type Data[Q any, T any] struct {
	List  []T   `json:"list"`
	Total int64 `json:"total"`
	Query Q     `json:"query"`
}

// Deprecated: use types.CursorQueryResult instead
type CursorData[Q any, T any] struct {
	List       []T       `json:"list"`
	Query      Q         `json:"query"`
	NextCursor *types.ID `json:"nextCursor,omitempty"`
}

type Sortable interface {
	GetSort() string
	GetDirection() Direction
}

type SortOption[T ~string] struct {
	Sort      T         `json:"sort,omitempty" query:"sort,omitempty" validate:"omitempty"`
	Direction Direction `json:"direction,omitempty" query:"direction,omitempty" validate:"omitempty"`
}

func (s SortOption[T]) GetSort() string {
	return string(s.Sort)
}

func (s SortOption[T]) GetDirection() Direction {
	return s.Direction
}

func (s SortOption[T]) Convert(c func(T) string) Sortable {
	return &sortable{sort: c(s.Sort), direction: s.Direction}
}

func (s SortOption[T]) Validate() errx.Error {
	if s.GetSort() == "" {
		return nil
	}
	if v, ok := any(s.Sort).(validation.Validatable); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	if ok := s.Direction.Enum().Contains(s.Direction); !ok {
		return errx.Validation.WithMsg("sort direction is invalid").Err()
	}
	return nil
}

type sortable struct {
	sort      string
	direction Direction
}

func (s *sortable) GetSort() string {
	return s.sort

}
func (s *sortable) GetDirection() Direction {
	return s.direction
}

type Direction string

func (d Direction) Enum() types.Enum {
	return types.RegisterEnum(DirectionAsc, DirectionDesc)
}

const (
	DirectionAsc  Direction = "asc"
	DirectionDesc Direction = "desc"
)
