package table

// xEdge represents the left and right X-coordinates of a text element.
type xEdge struct {
	left  float64
	right float64
}

// xCluster represents a group of overlapping X-ranges forming a column.
type xCluster struct {
	left  float64
	right float64
}
