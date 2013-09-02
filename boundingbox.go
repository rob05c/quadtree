package quadtree

type BoundingBox struct {
	Center        Point
	HalfDimension Point
}

func (b *BoundingBox) Contains(p *Point) bool {
	return p.X >= b.Center.X-b.HalfDimension.X &&
		p.X <= b.Center.X+b.HalfDimension.X &&
		p.Y >= b.Center.Y-b.HalfDimension.Y &&
		p.Y <= b.Center.Y+b.HalfDimension.Y
}

func (b *BoundingBox) Intersects(other *BoundingBox) bool {
	return b.Center.X+b.HalfDimension.X > other.Center.X-other.HalfDimension.X &&
		b.Center.X-b.HalfDimension.X < other.Center.X+other.HalfDimension.X &&
		b.Center.Y+b.HalfDimension.Y > other.Center.Y-other.HalfDimension.Y &&
		b.Center.Y-b.HalfDimension.Y < other.Center.Y+other.HalfDimension.Y
}
