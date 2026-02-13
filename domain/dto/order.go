package dto

type direction string

const (
	DESC direction = "DESC"
	ASC  direction = "ASC"
)

type OrderParam struct {
	Column    string
	Direction direction
}
