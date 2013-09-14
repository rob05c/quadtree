/*
Package quadtree implements a threadsafe lock-free quadtree

Currently, it only stores points, not ancillary data.
This could be trivially changed by adding a variable to PointListNode.
*/
package quadtree

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"unsafe"
)

type Quadtree struct {
	Boundary *BoundingBox
	Points   *PointList
	Nw       *Quadtree
	Ne       *Quadtree
	Sw       *Quadtree
	Se       *Quadtree
}

func New(boundingBox *BoundingBox, capacity int) *Quadtree {
	return &Quadtree{Boundary: boundingBox, Points: &PointList{Capacity: capacity}}
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
	// we don't need to compare. We know it needs set at nil now; if someone else set it first, setting again doesn't hurt.
	// this does need to be atomic, however. Else, Query() might read a pointer which was half-set to nil
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points)), nil)
}

// for the createDir funcs, we don't need to check the value of the CAS; if it fails, someone else succeeded, so we just continue
func (q *Quadtree) createNw() {
	quadrant := q.createQuadrant(Point{q.Boundary.Center.X - q.Boundary.HalfDimension.X/2.0, q.Boundary.Center.Y - q.Boundary.HalfDimension.Y/2.0})
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Nw)), nil, unsafe.Pointer(quadrant))
}

func (q *Quadtree) createNe() {
	quadrant := q.createQuadrant(Point{q.Boundary.Center.X + q.Boundary.HalfDimension.X/2.0, q.Boundary.Center.Y - q.Boundary.HalfDimension.Y/2.0})
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Ne)), nil, unsafe.Pointer(quadrant))
}

func (q *Quadtree) createSw() {
	quadrant := q.createQuadrant(Point{q.Boundary.Center.X - q.Boundary.HalfDimension.X/2.0, q.Boundary.Center.Y + q.Boundary.HalfDimension.Y/2.0})
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Sw)), nil, unsafe.Pointer(quadrant))
}

func (q *Quadtree) createSe() {
	quadrant := q.createQuadrant(Point{q.Boundary.Center.X + q.Boundary.HalfDimension.X/2.0, q.Boundary.Center.Y + q.Boundary.HalfDimension.Y/2.0})
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Se)), nil, unsafe.Pointer(quadrant))
}

func (q *Quadtree) createQuadrant(center Point) *Quadtree {
	// this is important. Otherwise, q.Points could be changed by another thread
	points := q.Points
	if points == nil {
		return nil // this is ok, because if q.Points became nil, the caller's CAS will fail
	}
	return &Quadtree{
		Boundary: &BoundingBox{center, Point{q.Boundary.HalfDimension.X / 2.0, q.Boundary.HalfDimension.Y / 2.0}},
		Points:   &PointList{Capacity: points.Capacity},
	}
}
