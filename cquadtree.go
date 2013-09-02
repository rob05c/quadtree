/*
Package quadtree implements a quadtree

Currently, it only stores points, not ancillary data. This could be trivially changed by adding another variable to the Point class, or changing the Quadtree struct to contain a map rather than slice of Points, or changing the Quadtree struct to contain a PointWrapper which contains a point and a data value. 

quadtree is not currently safe for concurrent use.
*/
package quadtree

import (
	"fmt"
	"strconv"
	"unsafe"
	"sync/atomic"
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
	Points   *[]Point
	Nw       *Quadtree
	Ne       *Quadtree
	Sw       *Quadtree
	Se       *Quadtree
}

func (q *Quadtree) Insert(p Point) bool {
	// we don't need to check the boundary within the CAS loop, because it can't change.
	// if the quadtree were changed to allow changing the Boundary, this would no longer be threadsafe.
	if !q.Boundary.Contains(&p) {
		return false
	}

	for {
		// the value we start working with
		oldPoints := q.Points
		// if at any point in our attempts to add the point, the length becomes the capacity, break so we can subdivide if necessary and add to a subtree
		if len(*oldPoints) >= cap(*oldPoints) {
			break
		}
		newPoints := append(*oldPoints, p)
		// if the working value is the same, set the new slice with our point
		ok := atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points)), unsafe.Pointer(oldPoints), unsafe.Pointer(&newPoints))
		if ok {
			// the CAS succeeded, our point was added, return success
			return true
		}
		fmt.Println("CAS Insert failed: len(points): " + strconv.Itoa(len(newPoints)))
		// if the working value changed underneath us, loop and try again
	}

	// If we get here, we broke the loop because the length exceeds the capacity. 
	// We must now Subdivide if necessary, and add the point to the proper subtree

	// at this point, with the above CAS, even if we simply mutex the Subdivide(), we will have achieved amortized lock-free time.

	// subdivide is threadsafe. The function itself does CAS
	q.subdivide()

	// These inserts are themselves threadsafe. Therefore, we don't need to do any special CAS work here.
	return q.Nw.Insert(p) ||
		q.Ne.Insert(p) ||
		q.Sw.Insert(p) ||
		q.Se.Insert(p)
}

// helper function of Quadtree.insert()
// subdivides the tree into quadrants. 
// This should be called when the capacity is exceeded.
func (q *Quadtree) subdivide() {
	if *q.Points == nil {
		return
	}
	if q.Nw == nil {
		q.createNw()
	}
	if q.Ne == nil {
		q.createNe()
	}
	if q.Sw == nil {
		q.createSw()
	}
	if q.Se == nil {
		q.createSe()
	}
	if *q.Points != nil {
		q.disperse()
	}
}

// helper function for Quadtree.subdivide()
//
// places all points in the tree in the appropriate quadrant, 
// and clears the points of this tree.
func (q *Quadtree) disperse() {
	for {
		oldPoints := q.Points
		if len(*oldPoints) == 0 {
			break
		}
		p := (*oldPoints)[0]
		newPoints := oldPoints
		if len(*oldPoints) == 1 {
			*newPoints = nil
		} else {
			*newPoints = (*oldPoints)[1:]
		}
		ok := atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points)), unsafe.Pointer(oldPoints), unsafe.Pointer(newPoints))
		if !ok {
			fmt.Println("CAS disperse failed: len(points): " + strconv.Itoa(len(*newPoints)))
			continue
		}
		ok = q.Nw.Insert(p) || q.Ne.Insert(p) || q.Sw.Insert(p) || q.Se.Insert(p)
		// @todo remove when quadtree is finished. Useful for debugging, but it slows things down
		if !ok {
			panic("quadtree contained a point outside boundary")
		}
	}
	*q.Points = nil
}


// helper function for Quadtree.createDir() functions
// creates a quadrant of the current Quadtree, with the given center
func (q *Quadtree) createQuadrant(center Point) *Quadtree {
	points := make([]Point, 0, cap(*q.Points))
	return &Quadtree{
		Boundary: &BoundingBox{
			Center: center,
			HalfDimension: Point{q.Boundary.HalfDimension.X / 2.0, q.Boundary.HalfDimension.Y / 2.0},
		},
		Points: &points,
	}
}

// helper function for Quadtree.subdivide()
// creates the Nw quadrant of the tree
func (q *Quadtree) createNw() {
	center := Point{
		X: q.Boundary.Center.X - q.Boundary.HalfDimension.X/2.0,
		Y: q.Boundary.Center.Y - q.Boundary.HalfDimension.Y/2.0,
	}
	quadrant := q.createQuadrant(center)
	// we don't need to check if the CAS fails - if it fails, someone else already created the quadrant
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Nw)), nil, unsafe.Pointer(quadrant))
}

// helper function for Quadtree.subdivide()
// creates the Ne quadrant of the tree
func (q *Quadtree) createNe() {
	center := Point{
		X: q.Boundary.Center.X + q.Boundary.HalfDimension.X/2.0,
		Y: q.Boundary.Center.Y - q.Boundary.HalfDimension.Y/2.0,
	}
	quadrant := q.createQuadrant(center)
	// we don't need to check if the CAS fails - if it fails, someone else already created the quadrant
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Ne)), nil, unsafe.Pointer(quadrant))
}

// helper function for Quadtree.subdivide()
// creates the Sw quadrant of the tree
func (q *Quadtree) createSw() {
	center := Point{
		X: q.Boundary.Center.X - q.Boundary.HalfDimension.X/2.0,
		Y: q.Boundary.Center.Y + q.Boundary.HalfDimension.Y/2.0,
	}
	quadrant := q.createQuadrant(center)
	// we don't need to check if the CAS fails - if it fails, someone else already created the quadrant
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Sw)), nil, unsafe.Pointer(quadrant))
}

// helper function for Quadtree.subdivide()
// creates the Se quadrant of the tree
func (q *Quadtree) createSe() {
	center := Point{
		X: q.Boundary.Center.X + q.Boundary.HalfDimension.X/2.0,
		Y: q.Boundary.Center.Y + q.Boundary.HalfDimension.Y/2.0,
	}
	quadrant := q.createQuadrant(center)
	// we don't need to check if the CAS fails - if it fails, someone else already created the quadrant
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Se)), nil, unsafe.Pointer(quadrant))
}

func (q *Quadtree) QueryRange(b *BoundingBox) []Point {
	var points []Point
	if !q.Boundary.Intersects(b) {
		return nil
	}
	for _, point := range *q.Points {
		if b.Contains(&point) {
			points = append(points, point)
		}
	}
	if q.Nw != nil {
		points = append(points, q.Nw.QueryRange(b)...)
	}
	if q.Ne != nil {
		points = append(points, q.Ne.QueryRange(b)...)
	}
	if q.Sw != nil {
		points = append(points, q.Sw.QueryRange(b)...)
	}
	if q.Se != nil {
		points = append(points, q.Se.QueryRange(b)...)
	}
	return points
}

func NewQuadTree(b *BoundingBox, capacity int) *Quadtree {
	points := make([]Point, 0, capacity)
	return &Quadtree {
		Boundary: b,
		Points: &points,
//		Nw: &Quadtree{},
//		Ne: &Quadtree{},
//		Sw: &Quadtree{},
//		Se: &Quadtree{},
	}
}
