package gamedata

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/s22678/statki/lib"
)

var (
	heatmap = map[string]int{}
	coords  = []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
)

func AddShotToHeatMap(coord string) {
	heatmap[coord] += 1
}

func GetHeatMap() map[string]int {
	return heatmap
}

func DisplayHeatMap() {
	fmt.Println()
	heat := [10][10]int{}
	LoadHeatMap()

	for k, v := range heatmap {
		x, y, _ := lib.CoordToIndex(k)
		heat[x][y] = v
	}

	for row := 0; row < 12; row++ {
		for column := 0; column < 13; column++ {
			// print vertical values 1-10
			if column == 0 && row > 1 {
				fmt.Print(row - 1)
				if row <= 10 {
					fmt.Print("  ")
				} else {
					fmt.Print(" ")
				}
			}

			// print horizontal values A-J
			if row == 0 {
				if column == 0 {
					fmt.Print("   ")
				}
				if column > 2 {
					fmt.Print(coords[column-3], " ")
				}
			}

			if column >= 3 && row >= 2 {
				x := column - 3
				y := row - 2
				fmt.Print(heat[x][y], "|")
			}
		}
		fmt.Println()
	}
	fmt.Println()
	fmt.Println()
}

func LoadHeatMap() error {
	data, err := os.ReadFile("heatmap.json")
	if err != nil {
		return err
	}
	if len(data) > 0 {
		err = json.Unmarshal(data, &heatmap)
	}
	return err
}

func SaveHeatMap() error {
	jsonStr, err := json.MarshalIndent(heatmap, "", " ")
	if err != nil {
		return err
	}
	err = os.WriteFile("heatmap.json", jsonStr, 0644)
	return err
}
