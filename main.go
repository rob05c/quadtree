package main

import (
	"fmt"
	"strconv"
	"math/rand"
	"time"
	"github.com/robert-butts/quadtree"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	nodeCapacity := 4
	qt := NewQuadTree(&BoundingBox {
		Center: Point { 100.0, 100.0 },
		HalfDimension: Point { 50.0, 50.0,},
	}, nodeCapacity)

	pointsToInsert := 10000000
	start := time.Now()
	for i := 0; i != pointsToInsert; i++ {
		p := Point{rand.Float64() * 100.0 + 50.0, rand.Float64() * 100.0 + 50.0}
		if !qt.Insert(p) {
		}
	}
	end := time.Now()
	elapsed := end.Sub(start)
	fmt.Println("inserted " + strconv.Itoa(pointsToInsert) + " points in " + elapsed.String() + ".")
/*
	qr := &BoundingBox {
		Center: Point { 75.0, 75.0 },
		HalfDimension: Point { 5.0, 5.0,},
	}
	points := qt.QueryRange(qr)
	fmt.Println("found " + strconv.Itoa(len(points)) + " points")
*/
//	for _, point := range points {
//		fmt.Println(point.String())
//	}
}
