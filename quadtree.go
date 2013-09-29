package quadtree

type Quadtree interface {
	Insert(p *Point) bool
	Query(b *BoundingBox) []Point
	Boundary() *BoundingBox
}

// lock-free is the default New
func New(boundingBox *BoundingBox, capacity int) Quadtree {
	return NewLockFree(boundingBox, capacity)
}