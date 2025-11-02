package resolvers

import (
	"fmt"
	"math"
)

func paginationValue(name string, value int) (int32, error) {
	if value < 0 {
		return 0, fmt.Errorf("%s must be non-negative", name)
	}
	if value > math.MaxInt32 {
		return 0, fmt.Errorf("%s exceeds max int32 (%d)", name, math.MaxInt32)
	}
	return int32(value), nil
}
