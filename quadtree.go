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

type BoundingBox struct {
	Center        Point
	HalfDimension Point
}

func (b *BoundingBox) Contains(p *Point) bool {
	return p.X >= b.Center.X-b.HalfDimension.X &&
		p.X <= b.Center.X+b.HalfDimension.X &&
		p.Y >= b.Center.Y-b.HalfDimension.Y &&
		p.Y <= b.Center.Y+b.HalfDimension.Y
}

func (b *BoundingBox) Intersects(other *BoundingBox) bool {
	return b.Center.X+b.HalfDimension.X > other.Center.X-other.HalfDimension.X &&
		b.Center.X-b.HalfDimension.X < other.Center.X+other.HalfDimension.X &&
		b.Center.Y+b.HalfDimension.Y > other.Center.Y-other.HalfDimension.Y &&
		b.Center.Y-b.HalfDimension.Y < other.Center.Y+other.HalfDimension.Y
}

type Quadtree struct {
	Boundary *BoundingBox
	Points   []Point
	Nw       *Quadtree
	Ne       *Quadtree
	Sw       *Quadtree
	Se       *Quadtree
}

func (q *Quadtree) Insert(p Point) bool {
	if !q.Boundary.Contains(&p) {
		return false
	}
	if len(q.Points) < cap(q.Points) {
		q.Points = append(q.Points, p)
		return true
	}
	if q.Nw == nil {
		q.Subdivide()
	}
	return q.Nw.Insert(p) ||
		q.Ne.Insert(p) ||
		q.Sw.Insert(p) ||
		q.Se.Insert(p)
}

func (q *Quadtree) Subdivide() {
	q.Nw = &Quadtree{
		Boundary: &BoundingBox{
			Center: Point{
				X: q.Boundary.Center.X - q.Boundary.HalfDimension.X/2.0,
				Y: q.Boundary.Center.Y - q.Boundary.HalfDimension.Y/2.0,
			},
			HalfDimension: Point{q.Boundary.HalfDimension.X / 2.0, q.Boundary.HalfDimension.Y / 2.0},
		},
		Points: make([]Point, 0, cap(q.Points)),
	}
	q.Ne = &Quadtree{
		Boundary: &BoundingBox{
			Center: Point{
				X: q.Boundary.Center.X - q.Boundary.HalfDimension.X/2.0,
				Y: q.Boundary.Center.Y + q.Boundary.HalfDimension.Y/2.0,
			},
			HalfDimension: Point{q.Boundary.HalfDimension.X / 2.0, q.Boundary.HalfDimension.Y / 2.0},
		},
		Points: make([]Point, 0, cap(q.Points)),
	}
	q.Sw = &Quadtree{
		Boundary: &BoundingBox{
			Center: Point{
				X: q.Boundary.Center.X + q.Boundary.HalfDimension.X/2.0,
				Y: q.Boundary.Center.Y - q.Boundary.HalfDimension.Y/2.0,
			},
			HalfDimension: Point{q.Boundary.HalfDimension.X / 2.0, q.Boundary.HalfDimension.Y / 2.0},
		},
		Points: make([]Point, 0, cap(q.Points)),
	}
	q.Se = &Quadtree{
		Boundary: &BoundingBox{
			Center: Point{
				X: q.Boundary.Center.X + q.Boundary.HalfDimension.X/2.0,
				Y: q.Boundary.Center.Y + q.Boundary.HalfDimension.Y/2.0,
			},
			HalfDimension: Point{q.Boundary.HalfDimension.X / 2.0, q.Boundary.HalfDimension.Y / 2.0},
		},
		Points: make([]Point, 0, cap(q.Points)),
	}
	for _, p := range q.Points {
		ok := q.Nw.Insert(p) || q.Ne.Insert(p) || q.Sw.Insert(p) || q.Se.Insert(p)
		if !ok {
			panic("quadtree contained a point outside boundary")
		}
	}
	q.Points = nil
}

func (q *Quadtree) QueryRange(b *BoundingBox) []Point {
	var points []Point
	if !q.Boundary.Intersects(b) {
		return nil
	}
	for _, point := range q.Points {
		if b.Contains(&point) {
			points = append(points, point)
		}
	}
	if q.Nw == nil {
		return points
	}
	points = append(points, q.Nw.QueryRange(b)...)
	points = append(points, q.Ne.QueryRange(b)...)
	points = append(points, q.Sw.QueryRange(b)...)
	points = append(points, q.Se.QueryRange(b)...)
	return points
}

func NewQuadTree(b *BoundingBox, capacity int) *Quadtree {
	return &Quadtree {
		Boundary: b,
		Points: make([]Point, 0, capacity),
	}
}
