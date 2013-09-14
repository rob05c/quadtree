package quadtree

import (
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func benchInsert(b *testing.B) *Quadtree {
	b.StopTimer()
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UnixNano())

	nodeCapacity := 4
	qt := New(&BoundingBox{
		Center:        Point{100.0, 100.0},
		HalfDimension: Point{50.0, 50.0},
	}, nodeCapacity)

	pointsToInsert := b.N

	threads := runtime.NumCPU()

	tpoints := pointsToInsert / threads
	done := make(chan bool)

	insertPoint := func() {
		for i := 0; i != tpoints; i++ {
			p := &Point{rand.Float64()*100.0 + 50.0, rand.Float64()*100.0 + 50.0}
			qt.Insert(p)
		}
		done <- true
	}
	b.StartTimer()
	b.ResetTimer()
	for i := 0; i != threads; i++ {
		go insertPoint()
	}
	for i := 0; i != threads; i++ {
		<-done
	}
	return qt
}

func BenchmarkInsert(b *testing.B) {
	benchInsert(b)
}

func BenchmarkQuery(b *testing.B) {
	b.StopTimer()
	qt := benchInsert(b)
	queries := b.N
	box := &BoundingBox{
		Center:        Point{100.0, 100.0},
		HalfDimension: Point{5.0, 5.0},
	}
	b.StartTimer()
	b.ResetTimer()
	for i := 0; i != queries; i++ {
		box.Center = Point{rand.Float64()*100.0 + 50.0, rand.Float64()*100.0 + 50.0}
		qt.Query(box)
	}
}
