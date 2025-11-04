package types

import (
	"github.com/tencent-go/pkg/errx"
)

type CursorQuery[T any] struct {
	Limit  int64 `json:"limit" query:"limit,omitempty" validate:"omitempty,min=1"`
	Cursor T     `json:"cursor,omitempty" query:"cursor,omitempty"`
}

type CursorQueryResult[Q any, T any] struct {
	List    []T  `json:"list"`
	Query   Q    `json:"query"`
	HasMore bool `json:"hasMore"`
	Cursor  any  `json:"cursor,omitempty"`
}

type Sortable interface {
	GetSort() string
	GetDirection() Direction
}

type SortOption[T ~string] struct {
	Sort      T         `json:"sort,omitempty" query:"sort,omitempty"`
	Direction Direction `json:"direction,omitempty" query:"direction,omitempty"`
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

func (d Direction) Enums() []any {
	return []any{"asc", "desc"}
}

func (d Direction) Validate() errx.Error {
	if d != DirectionAsc && d != DirectionDesc {
		return errx.Validation.WithMsgf("invalid direction %s", d).Err()
	}
	return nil
}

const (
	DirectionAsc  Direction = "asc"
	DirectionDesc Direction = "desc"
)
