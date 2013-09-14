package quadtree

type PointListNode struct {
	*Point
	Next *PointListNode
}

func NewPointListNode(point *Point, next *PointListNode) *PointListNode {
	return &PointListNode{
		Point: point,
		Next:  next,
	}
}

type PointList struct {
	Capacity int
	First    *PointListNode
	Length   int // this is a cache for speed; it could be calculated from the PointsList
}
