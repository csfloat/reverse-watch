package dto

type Direction string

const (
	DESC Direction = "DESC"
	ASC  Direction = "ASC"
)

type OrderParam struct {
	Column    string
	Direction Direction
}
