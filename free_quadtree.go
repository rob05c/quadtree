/*
Package quadtree implements a threadsafe lock-free quadtree

Currently, it only stores points, not ancillary data.
This could be trivially changed by adding a variable to PointListNode.

// @todo fix loads to use AtomicLoad. They are not threadsafe. It can be changed in the middle of a load
*/
package quadtree

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

type LockfreeQuadtree struct {
	boundary *BoundingBox
	Points   *PointList
	Nw       *LockfreeQuadtree
	Ne       *LockfreeQuadtree
	Sw       *LockfreeQuadtree
	Se       *LockfreeQuadtree
}

func NewLockFree(boundingBox *BoundingBox, capacity int) Quadtree {
	return &LockfreeQuadtree{boundary: boundingBox, Points: &PointList{Capacity: capacity}}
}

func (q *LockfreeQuadtree) Boundary() *BoundingBox {
	return q.boundary
}

func (q *LockfreeQuadtree) Insert(p *Point) bool {
	// we don't need to check the boundary within the CAS loop, because it can't change.
	// if the quadtree were changed to allow changing the boundary, this would no longer be threadsafe.
	if !q.boundary.Contains(p) {
		//		fmt.Println("insert outside boundary")
		return false
	}

	for {
		// the value we start working with
		oldPoints := (*PointList)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points))))
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
	points := (*PointList)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points))))
	if points != nil {
		q.subdivide()
	}

	// These inserts are themselves threadsafe. Therefore, we don't need to do any special CAS work here.
	ok := q.Nw.Insert(p) || q.Ne.Insert(p) || q.Sw.Insert(p) || q.Se.Insert(p)
	if !ok {
		fmt.Println("insert failed")
	}
	return ok
}

func (q *LockfreeQuadtree) Query(b *BoundingBox) []Point {
	var points []Point
	if !q.boundary.Intersects(b) {
		return nil
	}
	// this is important. It prevents the loop from segfaulting if a concurrent thread causes a split on this qt
	// note this is safe because q.Points is only ever set atomically. If it were not, this would be unsafe for concurrent use
	qPoints := (*PointList)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points))))
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
func (q *LockfreeQuadtree) subdivide() {
	points := (*PointList)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points))))
	if points == nil {
		return
	}
	q.createNw()
	q.createNe()
	q.createSw()
	q.createSe()
	q.disperse()
}

func (q *LockfreeQuadtree) disperse() {
	oldPoints := (*PointList)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points))))
	if oldPoints == nil {
		return
	}
	ok := atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points)), unsafe.Pointer(oldPoints), nil)
	if !ok {
		return // someone beat us to it
	}
	for oldPoints.First != nil {
		p := oldPoints.First.Point
		oldPoints.First = oldPoints.First.Next
		oldPoints.Length--
		ok = q.Nw.Insert(p) || q.Ne.Insert(p) || q.Sw.Insert(p) || q.Se.Insert(p)
		if !ok {
			panic("disperse point outside bounds")
		}
	}
}

// helper function for subdivide()
//
// places all points in the tree in the appropriate quadrant,
// and clears the points of this tree.
func (q *LockfreeQuadtree) disperse2() {
	for {
		oldPoints := (*PointList)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points))))
		if oldPoints == nil || oldPoints.Length == 0 {
			break
		}
		newPoints := *oldPoints
		p := *newPoints.First.Point
		newPoints.First = newPoints.First.Next
		newPoints.Length--
		ok := atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points)), unsafe.Pointer(oldPoints), unsafe.Pointer(&newPoints))
		if !ok {
			continue
		}

		ok = q.Nw.Insert(&p) || q.Ne.Insert(&p) || q.Sw.Insert(&p) || q.Se.Insert(&p)
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
func (q *LockfreeQuadtree) createNw() {
	quadrant := q.createQuadrant(Point{q.boundary.Center.X - q.boundary.HalfDimension.X/2.0, q.boundary.Center.Y - q.boundary.HalfDimension.Y/2.0})
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Nw)), nil, unsafe.Pointer(quadrant))
}

func (q *LockfreeQuadtree) createNe() {
	quadrant := q.createQuadrant(Point{q.boundary.Center.X + q.boundary.HalfDimension.X/2.0, q.boundary.Center.Y - q.boundary.HalfDimension.Y/2.0})
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Ne)), nil, unsafe.Pointer(quadrant))
}

func (q *LockfreeQuadtree) createSw() {
	quadrant := q.createQuadrant(Point{q.boundary.Center.X - q.boundary.HalfDimension.X/2.0, q.boundary.Center.Y + q.boundary.HalfDimension.Y/2.0})
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Sw)), nil, unsafe.Pointer(quadrant))
}

func (q *LockfreeQuadtree) createSe() {
	quadrant := q.createQuadrant(Point{q.boundary.Center.X + q.boundary.HalfDimension.X/2.0, q.boundary.Center.Y + q.boundary.HalfDimension.Y/2.0})
	atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Se)), nil, unsafe.Pointer(quadrant))
}

func (q *LockfreeQuadtree) createQuadrant(center Point) *LockfreeQuadtree {
	// this is important. Otherwise, q.Points could be changed by another thread
	points := (*PointList)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&q.Points))))
	if points == nil {
		return nil // this is ok, because if q.Points became nil, the caller's CAS will fail
	}
	return &LockfreeQuadtree{
		boundary: &BoundingBox{center, Point{q.boundary.HalfDimension.X / 2.0, q.boundary.HalfDimension.Y / 2.0}},
		Points:   &PointList{Capacity: points.Capacity},
	}
}
