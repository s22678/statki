package lib

import (
	"strconv"

	"github.com/takuoki/clmconv"
)

// Modified coordinates like A1, B2 to (int, int) pair. A1 = (1,1), B3 = (2,3) etc.
func CoordToIndex(coord string) (setx int, sety int, err error) {
	setx, err = clmconv.Atoi(coord[:1])
	if err != nil {
		return 0, 0, err
	}
	if len(coord) == 2 {
		sety, err = strconv.Atoi(coord[1:2])
		if err != nil {
			return 0, 0, err
		}
	}
	if len(coord) >= 3 {
		sety, err = strconv.Atoi(coord[1:3])
		if err != nil {
			return 0, 0, err
		}
	}
	return setx, sety - 1, nil
}
