/*
Package quadtree implements a threadsafe lock-free quadtree

Currently, it only stores points, not ancillary data. 
This could be trivially changed by adding a variable to PointListNode.
*/
package quadtree

import (
	"fmt"
	"strconv"
	"unsafe"
	"sync/atomic"
)

type Quadtree struct {
	Boundary *BoundingBox
	Points   *PointList
	Nw       *Quadtree
	Ne       *Quadtree
	Sw       *Quadtree
	Se       *Quadtree
}

func New(b *BoundingBox, capacity int) *Quadtree {
	return &Quadtree {
		Boundary: b,
		Points: NewPointList(capacity),
	}
}

func (q *Quadtree) Insert(p *Point) bool {
	// we don't need to check the boundary within the CAS loop, because it can't change.
	// if the quadtree were changed to allow changing the Boundary, this would no longer be threadsafe.
	if !q.Boundary.Contains(p) {
//		fmt.Println("insert outside boundary")
		return false
	}
	for {
		// the value we start working with
		oldPoints := q.Points
		// if at any point in our attempts to add the point, the length becomes the capacity, break so we can subdivide if necessary and add to a subtree
		if oldPoints == nil || oldPoints.Length >= oldPoints.Capacity {
			break
		}
		newPoints := *oldPoints
		newPoints.First = NewPointListNode(p, newPoints.First)
		newPoints.Length++
		// if the working value is the same, set the new slice with our point
		ok := atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points)), unsafe.Pointer(oldPoints), unsafe.Pointer(&newPoints))
		if ok {
			// the CAS succeeded, our point was added, return success
			return true
		}
		// debug
//		fmt.Println("CAS Insert failed: len(points): " + strconv.Itoa(newPoints.Length))
		// if the working value changed underneath us, loop and try again
	}

	// If we get here, we broke the loop because the length exceeds the capacity. 
	// We must now Subdivide if necessary, and add the point to the proper subtree

	// at this point, with the above CAS, even if we simply mutex the Subdivide(), we will have achieved amortized lock-free time.

	// subdivide is threadsafe. The function itself does CAS
	if q.Points != nil {
		q.subdivide()
	}

	// These inserts are themselves threadsafe. Therefore, we don't need to do any special CAS work here.
	return q.Nw.Insert(p) ||
		q.Ne.Insert(p) ||
		q.Sw.Insert(p) ||
		q.Se.Insert(p)
}

func (q *Quadtree) Query(b *BoundingBox) []Point {
	var points []Point
	if !q.Boundary.Intersects(b) {
		return nil
	}
	// this is important. It prevents the loop from segfaulting if a concurrent thread causes a split on this qt
	// note this is safe because q.Points is only ever set atomically. If it were not, this would be unsafe for concurrent use
	qPoints := q.Points
	if qPoints != nil {
		for node := qPoints.First; node != nil; node = node.Next {
			if b.Contains(node.Point) {
				points = append(points, *node.Point)
			}
		}
	}
	if q.Nw != nil {
		points = append(points, q.Nw.Query(b)...)
	}
	if q.Ne != nil {
		points = append(points, q.Ne.Query(b)...)
	}
	if q.Sw != nil {
		points = append(points, q.Sw.Query(b)...)
	}
	if q.Se != nil {
		points = append(points, q.Se.Query(b)...)
	}
	return points
}

// helper function of Insert()
// subdivides the tree into quadrants. 
// This should be called when the capacity is exceeded.
func (q *Quadtree) subdivide() {
	if q.Points == nil {
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
	if q.Points != nil {
		q.disperse()
	}
}

// helper function for subdivide()
//
// places all points in the tree in the appropriate quadrant, 
// and clears the points of this tree.
func (q *Quadtree) disperse() {
	for {
		oldPoints := q.Points
		if oldPoints == nil || oldPoints.Length == 0 {
			break
		}
		newPoints := *oldPoints
		// debug
		if newPoints.First == nil {
			fmt.Println("nil first with " + strconv.Itoa(oldPoints.Length))
		}
		p := newPoints.First.Point
		newPoints.First = newPoints.First.Next
		newPoints.Length--
		ok := atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points)), unsafe.Pointer(oldPoints), unsafe.Pointer(&newPoints))
		if !ok {
			// debug
//			fmt.Println("CAS disperse failed: len(points): " + strconv.Itoa(newPoints.Length))
			continue
		}
		ok = q.Nw.Insert(p) || q.Ne.Insert(p) || q.Sw.Insert(p) || q.Se.Insert(p)
		// debug
		if !ok {
			panic("quadtree contained a point outside boundary")
		}
	}
	q.Points = nil
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

// helper function for Quadtree.createDir() functions
// creates a quadrant of the current Quadtree, with the given center
func (q *Quadtree) createQuadrant(center Point) *Quadtree {
	return &Quadtree{
		Boundary: &BoundingBox{
			Center: center,
			HalfDimension: Point{q.Boundary.HalfDimension.X / 2.0, q.Boundary.HalfDimension.Y / 2.0},
		},
		Points: NewPointList(q.Points.Capacity),
	}
}
