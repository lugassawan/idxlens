package layout

import (
	"math"
	"sort"

	"github.com/lugassawan/idxlens/internal/pdf"
)

const (
	// defaultLineThreshold is the fraction of dominant font size used to
	// cluster text elements into the same line. Elements whose Y coordinates
	// differ by less than threshold * dominantFontSize are grouped together.
	// A value of 0.5 accommodates slight baseline variations without merging
	// adjacent text lines in financial statements.
	defaultLineThreshold = 0.5
	regionXTolerance     = 5.0
	spaceGapRatio        = 0.3
	fallbackWidthRatio   = 0.5

	// columnGapMultiplier is the factor of average character width above
	// which a horizontal gap between elements is treated as a column
	// boundary. Gaps exceeding this threshold produce a tab character in
	// the joined Text field so that downstream layers (L2) can detect
	// column boundaries without relying solely on element coordinates.
	columnGapMultiplier = 3.0
)

type fontKey struct {
	name string
	size float64
}

// NewAnalyzer creates a layout analyzer with default settings.
func NewAnalyzer() Analyzer {
	return &analyzer{
		lineThreshold: defaultLineThreshold,
	}
}

type analyzer struct {
	lineThreshold float64
}

func (a *analyzer) Analyze(page pdf.Page) (LayoutPage, error) {
	if len(page.Elements) == 0 {
		return LayoutPage{
			Number: page.Number,
			Size:   page.Size,
		}, nil
	}

	dominantFontSize := findDominantFontSize(page.Elements)
	clusters := a.clusterByY(page.Elements, dominantFontSize)
	lines := assembleLines(clusters, dominantFontSize)
	sortLinesTopToBottom(lines)
	regions := detectRegions(lines)

	return LayoutPage{
		Number:  page.Number,
		Size:    page.Size,
		Lines:   lines,
		Regions: regions,
	}, nil
}

func findDominantFontSize(elements []pdf.TextElement) float64 {
	counts := make(map[float64]int)
	for _, e := range elements {
		counts[e.FontSize]++
	}

	var dominant float64
	var maxCount int

	for size, count := range counts {
		if count > maxCount || (count == maxCount && size > dominant) {
			dominant = size
			maxCount = count
		}
	}

	return dominant
}

func (a *analyzer) clusterByY(elements []pdf.TextElement, dominantFontSize float64) [][]pdf.TextElement {
	if len(elements) == 0 {
		return nil
	}

	threshold := a.lineThreshold * dominantFontSize

	sorted := make([]pdf.TextElement, len(elements))
	copy(sorted, elements)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Bounds.Y1 > sorted[j].Bounds.Y1
	})

	var clusters [][]pdf.TextElement
	cluster := []pdf.TextElement{sorted[0]}

	for i := 1; i < len(sorted); i++ {
		prevY := cluster[0].Bounds.Y1
		currY := sorted[i].Bounds.Y1

		if math.Abs(prevY-currY) < threshold {
			cluster = append(cluster, sorted[i])
		} else {
			clusters = append(clusters, cluster)
			cluster = []pdf.TextElement{sorted[i]}
		}
	}

	clusters = append(clusters, cluster)

	return clusters
}

func assembleLines(clusters [][]pdf.TextElement, dominantFontSize float64) []TextLine {
	lines := make([]TextLine, 0, len(clusters))

	for _, cluster := range clusters {
		line := assembleLine(cluster, dominantFontSize)
		lines = append(lines, line)
	}

	return lines
}

func assembleLine(elements []pdf.TextElement, dominantFontSize float64) TextLine {
	sort.Slice(elements, func(i, j int) bool {
		return elements[i].Bounds.X1 < elements[j].Bounds.X1
	})

	avgCharWidth := estimateAvgCharWidth(elements, dominantFontSize)
	text := buildLineText(elements, avgCharWidth)
	bounds := computeUnionBounds(elements)
	fontName, fontSize := dominantFont(elements)

	return TextLine{
		Text:     text,
		Elements: elements,
		Bounds:   bounds,
		FontName: fontName,
		FontSize: fontSize,
	}
}

func estimateAvgCharWidth(elements []pdf.TextElement, dominantFontSize float64) float64 {
	var totalWidth float64
	var totalChars int

	for _, e := range elements {
		charCount := len([]rune(e.Text))
		if charCount > 0 {
			totalWidth += e.Bounds.X2 - e.Bounds.X1
			totalChars += charCount
		}
	}

	if totalChars == 0 {
		return dominantFontSize * fallbackWidthRatio
	}

	return totalWidth / float64(totalChars)
}

func buildLineText(elements []pdf.TextElement, avgCharWidth float64) string {
	if len(elements) == 0 {
		return ""
	}

	columnThreshold := avgCharWidth * columnGapMultiplier

	var result []byte
	result = append(result, elements[0].Text...)

	for i := 1; i < len(elements); i++ {
		gap := elements[i].Bounds.X1 - elements[i-1].Bounds.X2

		switch {
		case gap >= columnThreshold:
			result = append(result, '\t')
		case gap > avgCharWidth*spaceGapRatio:
			result = append(result, ' ')
		}

		result = append(result, elements[i].Text...)
	}

	return string(result)
}

func computeUnionBounds(elements []pdf.TextElement) pdf.Rect {
	if len(elements) == 0 {
		return pdf.Rect{}
	}

	bounds := elements[0].Bounds

	for _, e := range elements[1:] {
		bounds.X1 = math.Min(bounds.X1, e.Bounds.X1)
		bounds.Y1 = math.Min(bounds.Y1, e.Bounds.Y1)
		bounds.X2 = math.Max(bounds.X2, e.Bounds.X2)
		bounds.Y2 = math.Max(bounds.Y2, e.Bounds.Y2)
	}

	return bounds
}

func dominantFont(elements []pdf.TextElement) (string, float64) {
	if len(elements) == 0 {
		return "", 0
	}

	counts := make(map[fontKey]int)

	for _, e := range elements {
		key := fontKey{name: e.FontName, size: e.FontSize}
		counts[key]++
	}

	var bestKey fontKey
	var maxCount int

	for key, count := range counts {
		if count > maxCount {
			bestKey = key
			maxCount = count
		}
	}

	return bestKey.name, bestKey.size
}

func sortLinesTopToBottom(lines []TextLine) {
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Bounds.Y1 > lines[j].Bounds.Y1
	})
}

func detectRegions(lines []TextLine) []Region {
	if len(lines) == 0 {
		return nil
	}

	var regions []Region
	currentLines := []TextLine{lines[0]}

	for i := 1; i < len(lines); i++ {
		if linesShareRegion(currentLines[len(currentLines)-1], lines[i]) {
			currentLines = append(currentLines, lines[i])
		} else {
			regions = append(regions, buildRegion(currentLines))
			currentLines = []TextLine{lines[i]}
		}
	}

	regions = append(regions, buildRegion(currentLines))

	return regions
}

func linesShareRegion(prev, curr TextLine) bool {
	sameIndent := math.Abs(prev.Bounds.X1-curr.Bounds.X1) < regionXTolerance
	sameFont := prev.FontName == curr.FontName && prev.FontSize == curr.FontSize

	return sameIndent && sameFont
}

func buildRegion(lines []TextLine) Region {
	bounds := lines[0].Bounds

	for _, line := range lines[1:] {
		bounds.X1 = math.Min(bounds.X1, line.Bounds.X1)
		bounds.Y1 = math.Min(bounds.Y1, line.Bounds.Y1)
		bounds.X2 = math.Max(bounds.X2, line.Bounds.X2)
		bounds.Y2 = math.Max(bounds.Y2, line.Bounds.Y2)
	}

	return Region{
		Lines:  lines,
		Bounds: bounds,
	}
}
