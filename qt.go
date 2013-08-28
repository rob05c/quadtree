package main

import (
	"fmt"
)

type Point struct {
	X float64
	Y float64
}

type BoundingBox struct {
	Center        Point
	HalfDimension Point
}

func (b BoundingBox) Contains(p Point) bool {
	return p.X >= b.Center.X-b.HalfDimension.X &&
		p.X <= b.Center.X+b.HalfDimension.X &&
		p.Y >= b.Center.Y-b.HalfDimension.Y &&
		p.Y <= b.Center.Y+b.HalfDimension.Y
}

func (b BoundingBox) Intersects(other BoundingBox) bool {
	return b.Center.X+b.HalfDimension.X > other.Center.X-other.HalfDimension.X &&
		b.Center.X-b.HalfDimension.X < other.Center.X+other.HalfDimension.X &&
		b.Center.Y+b.HalfDimension.Y > other.Center.Y-other.HalfDimension.Y &&
		b.Center.Y-b.HalfDimension.Y < other.Center.Y+other.HalfDimension.Y
}

const NodeCapacity = 4

type Quadtree struct {
	Boundary *BoundingBox
	Nw       *Quadtree
	Ne       *Quadtree
	Sw       *Quadtree
	Se       *Quadtree
	Points   []Point
}

func (q Quadtree) Insert(p Point) bool {
	if !q.Boundary.Contains(p) {
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

func (q Quadtree) Subdivide() {
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

func (q Quadtree) QueryRange(b BoundingBox) {

}

func main() {
	fmt.Println("main")
}
