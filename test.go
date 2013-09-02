package main

import (
	"runtime"
	"fmt"
	"github.com/robert-butts/quadtree"
	"math/rand"
	"strconv"
	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UnixNano())
}

func main() {
	nodeCapacity := 4
	qt := quadtree.New(&quadtree.BoundingBox{
		Center:        quadtree.Point{100.0, 100.0},
		HalfDimension: quadtree.Point{50.0, 50.0},
	}, nodeCapacity)


	pointsToInsert := 10000000
	queries := 100
	threads := runtime.NumCPU()

	tpoints := pointsToInsert / threads
	done := make(chan bool)

	insertPoint := func() {
		for i := 0; i != tpoints; i++ {
			p := &quadtree.Point{rand.Float64()*100.0 + 50.0, rand.Float64()*100.0 + 50.0}
			qt.Insert(p)
		}
		done <- true
	}
	start := time.Now()
	for i := 0; i != threads; i++ {
		go insertPoint()
	}

	for i := 0; i != threads; i++ {
		<- done
	}
	end := time.Now()
	elapsed := end.Sub(start)
	fmt.Println("inserted " + strconv.Itoa(tpoints * threads) + " points in " + elapsed.String() + ".")

	found := 0
	start = time.Now()
	box := &quadtree.BoundingBox {
		Center: quadtree.Point { 100.0, 100.0 },
		HalfDimension: quadtree.Point { 5.0, 5.0,},
	}
	for i := 0; i != queries; i++ {
		box.Center = quadtree.Point{rand.Float64()*100.0 + 50.0, rand.Float64()*100.0 + 50.0}
		points := qt.Query(box)
		found += len(points)
	}
//	fmt.Println("found " + strconv.Itoa(len(points)) + " points")
	end = time.Now()
	elapsed = end.Sub(start)
	fmt.Println("queried " + strconv.Itoa(found) + " points via " + strconv.Itoa(queries) + " queries in " + elapsed.String() + ".")
	//	for _, point := range points {
	//		fmt.Println(point.String())
	//	}
}
