package quadtree

import (
	"math/rand"
	"runtime"
	"testing"
	"time"
)

/// note the number of points inserted is only approximately points
/// the actual number inserted is the closest power of threads
func testInsert(points int, threads int) (inserted int, tree *Quadtree) {
	runtime.GOMAXPROCS(threads / 4)
	rand.Seed(time.Now().UnixNano())
	nodeCapacity := 4
	qt := New(&BoundingBox{
		Center:        Point{100.0, 100.0},
		HalfDimension: Point{50.0, 50.0},
	}, nodeCapacity)
	tpoints := points / threads
	done := make(chan bool)
	insertPoint := func() {
		for i := 0; i != tpoints; i++ {
			p := &Point{rand.Float64()*100.0 + 50.0, rand.Float64()*100.0 + 50.0}
			qt.Insert(p)
		}
		done <- true
	}
	for i := 0; i != threads; i++ {
		go insertPoint()
	}
	for i := 0; i != threads; i++ {
		<-done
	}
	return tpoints * threads, qt
}

func TestInsertQuery(t *testing.T) {
	inserted, qt := testInsert(100000, 100)
	queried := len(qt.Query(qt.Boundary))
	if inserted != queried {
		t.Fail()
	}
}
