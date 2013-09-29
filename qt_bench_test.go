package quadtree

import (
	"math/rand"
	"runtime"
	"testing"
	"time"
)

const LowCapacity = 4
const HighCapacity = 1000

func newLockfree(capacity int) Quadtree {
	return NewLockFree(&BoundingBox{
		Center:        Point{100.0, 100.0},
		HalfDimension: Point{50.0, 50.0},
	}, capacity)
}
func newLockbased(capacity int) Quadtree {
	return NewLockFree(&BoundingBox{
		Center:        Point{100.0, 100.0},
		HalfDimension: Point{50.0, 50.0},
	}, capacity)
}
func benchInsert(b *testing.B, qt Quadtree) Quadtree {
	b.StopTimer()
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UnixNano())

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
func benchQuery(b *testing.B, qt Quadtree) {
	b.StopTimer()
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
func Benchmark_Insert_LowCapacity_LockFree(b *testing.B) {
	benchInsert(b,newLockfree(LowCapacity))
}
func Benchmark_Insert_LowCapacity_LockBased(b *testing.B) {
	benchInsert(b,newLockbased(LowCapacity))
}
func Benchmark_Insert_HighCapacity_LockFree(b *testing.B) {
	benchInsert(b,newLockfree(HighCapacity))
}
func Benchmark_Insert_HighCapacity_LockBased(b *testing.B) {
	benchInsert(b,newLockbased(HighCapacity))
}
func Benchmark_Query_LowCapacity_LockFree(b *testing.B) {
	benchQuery(b, benchInsert(b, newLockfree(LowCapacity)))
}
func Benchmark_Query_LowCapacity_LockBased(b *testing.B) {
	benchQuery(b, benchInsert(b, newLockbased(LowCapacity)))
}
func Benchmark_Query_HighCapacity_LockFree(b *testing.B) {
	benchQuery(b, benchInsert(b, newLockfree(HighCapacity)))
}
func Benchmark_Query_HighCapacity_LockBased(b *testing.B) {
	benchQuery(b, benchInsert(b, newLockbased(HighCapacity)))
}
