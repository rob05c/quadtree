/*
Package quadtree implements a threadsafe lock-free quadtree

Currently, it only stores points, not ancillary data.
This could be trivially changed by adding a variable to PointListNode.
*/
package quadtree

import (
	"fmt"
	"sync"
)

type LockbasedQuadtree struct {
	Points   *PointList
	Nw       *LockbasedQuadtree
	Ne       *LockbasedQuadtree
	Sw       *LockbasedQuadtree
	Se       *LockbasedQuadtree
	mutex    sync.RWMutex
	boundary *BoundingBox
}

func NewLockBased(boundingBox *BoundingBox, capacity int) Quadtree { 
	return &LockbasedQuadtree{boundary: boundingBox, Points: &PointList{Capacity: capacity}}
}

func (q *LockbasedQuadtree) Boundary() *BoundingBox {
	return q.boundary
}


func (q *LockbasedQuadtree) Insert(p *Point) bool {
	// if the quadtree were changed to allow changing the Boundary, this would no longer be threadsafe.
	if !q.boundary.Contains(p) {
		//		fmt.Println("insert outside boundary") // debug
		return false
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.Points != nil {
		q.Points.First = NewPointListNode(p, q.Points.First)
		q.Points.Length++
		return true
	}

	q.subdivide()
	ok := q.Nw.Insert(p) || q.Ne.Insert(p) || q.Sw.Insert(p) || q.Se.Insert(p)
	if !ok {
		fmt.Println("insert failed") // debug
	}
	return ok
}

func (q *LockbasedQuadtree) Query(b *BoundingBox) []Point {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	var points []Point
	if !q.boundary.Intersects(b) {
		return nil
	}
	if q.Points != nil {
		for node := q.Points.First; node != nil; node = node.Next {
			if b.Contains(node.Point) { // this can be ommitted if the capacity is small enough and absolute precision isn't required
				points = append(points, *node.Point)
			}
		}
		return points
	}

	points = append(points, q.Nw.Query(b)...)
	points = append(points, q.Ne.Query(b)...)
	points = append(points, q.Sw.Query(b)...)
	points = append(points, q.Se.Query(b)...)
	return points
}

// helper function of Insert()
// subdivides the tree into quadrants.
// This should be called when the capacity is exceeded.
func (q *LockbasedQuadtree) subdivide() {
	q.createNw()
	q.createNe()
	q.createSw()
	q.createSe()
	q.disperse()
}

func (q *LockbasedQuadtree) disperse() {
	for q.Points.First != nil {
		p := q.Points.First.Point
		q.Points.First = q.Points.First.Next
		q.Points.Length--
		ok := q.Nw.Insert(p) || q.Ne.Insert(p) || q.Sw.Insert(p) || q.Se.Insert(p)
		if !ok {
			panic("disperse point outside bounds")
		}
	}
	q.Points = nil
}

// for the createDir funcs, we don't need to check the value of the CAS; if it fails, someone else succeeded, so we just continue
func (q *LockbasedQuadtree) createNw() {
	q.Nw = q.createQuadrant(Point{q.boundary.Center.X - q.boundary.HalfDimension.X/2.0, q.boundary.Center.Y - q.boundary.HalfDimension.Y/2.0})
}

func (q *LockbasedQuadtree) createNe() {
	q.Ne = q.createQuadrant(Point{q.boundary.Center.X + q.boundary.HalfDimension.X/2.0, q.boundary.Center.Y - q.boundary.HalfDimension.Y/2.0})
}

func (q *LockbasedQuadtree) createSw() {
	q.Sw = q.createQuadrant(Point{q.boundary.Center.X - q.boundary.HalfDimension.X/2.0, q.boundary.Center.Y + q.boundary.HalfDimension.Y/2.0})
}

func (q *LockbasedQuadtree) createSe() {
	q.Se = q.createQuadrant(Point{q.boundary.Center.X + q.boundary.HalfDimension.X/2.0, q.boundary.Center.Y + q.boundary.HalfDimension.Y/2.0})
}

func (q *LockbasedQuadtree) createQuadrant(center Point) *LockbasedQuadtree {
	return &LockbasedQuadtree{
		boundary: &BoundingBox{center, Point{q.boundary.HalfDimension.X / 2.0, q.boundary.HalfDimension.Y / 2.0}},
		Points:   &PointList{Capacity: q.Points.Capacity},
	}
}
