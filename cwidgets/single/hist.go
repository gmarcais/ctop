package single

type IntHist struct {
	Val    int   // most current data point
	Data   []int // historical data points
	Labels []string
}

func NewIntHist(max int) *IntHist {
	return &IntHist{
		Data:   make([]int, max),
		Labels: make([]string, max),
	}
}

func (h *IntHist) Append(val int) {
	if len(h.Data) == cap(h.Data) {
		h.Data = append(h.Data[:0], h.Data[1:]...)
	}
	h.Val = val
	h.Data = append(h.Data, val)
}

// Grow adds empty history on the left, preserving all retained samples. Smaller
// viewports select a suffix instead of discarding history on every resize.
func (h *IntHist) Grow(size int) {
	h.Data = growHistory(h.Data, size)
	h.Labels = growHistory(h.Labels, size)
}

func growHistory[T any](data []T, size int) []T {
	if size <= len(data) {
		return data
	}
	grown := make([]T, size)
	copy(grown[size-len(data):], data)
	return grown
}

func visibleHistory[T any](data []T, width int) []T {
	if width < 0 {
		width = 0
	}
	if width < len(data) {
		return data[len(data)-width:]
	}
	return data
}

type DiffHist struct {
	*IntHist
	lastVal int
}

func NewDiffHist(max int) *DiffHist {
	return &DiffHist{NewIntHist(max), -1}
}

func (h *DiffHist) Append(val int) {
	if h.lastVal >= 0 { // skip append if this is the initial update
		diff := val - h.lastVal
		h.IntHist.Append(diff)
	}
	h.lastVal = val
}

type FloatHist struct {
	Val    float64   // most current data point
	Data   []float64 // historical data points
	Labels []string
}

func NewFloatHist(max int) FloatHist {
	return FloatHist{
		Data:   make([]float64, max),
		Labels: make([]string, max),
	}
}

func (h *FloatHist) Append(val float64) {
	if len(h.Data) == cap(h.Data) {
		h.Data = append(h.Data[:0], h.Data[1:]...)
	}
	h.Val = val
	h.Data = append(h.Data, val)
}

func (h *FloatHist) Grow(size int) {
	h.Data = growHistory(h.Data, size)
	h.Labels = growHistory(h.Labels, size)
}
