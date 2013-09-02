package quadtree

import (
	"strconv"
)

type Point struct {
	X float64
	Y float64
}

func (p *Point) String() string {
	return "[" + strconv.FormatFloat(p.X, 'f', -1, 64) + "," + strconv.FormatFloat(p.Y, 'f', -1, 64) + "]"
}
