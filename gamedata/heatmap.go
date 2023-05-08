package gamedata

import (
	"encoding/json"
	"fmt"
	"os"
)

var (
	heatmap = map[string]int{}
)

func AddShotToHeatMap(coord string) {
	heatmap[coord] += 1
}

func GetHeatMap() map[string]int {
	return heatmap
}

func DisplayHeatMap() {
	heat := [10][10]int{}
	LoadHeatMap()
	// keys := make([]string, len(heatmap))
	// i := 0
	for k, v := range heatmap {
		x, y, _ := CoordToIndex(k)
		heat[x][y] = v
	}

	for row := 0; row < 10; row++ {
		for column := 0; column < 12; column++ {
			if column == 10 {
				continue
			} else if column == 11 {
				fmt.Print(row - 1)
			} else {
				fmt.Print(heat[row][column], " ")
			}
		}
		fmt.Print("\n")
	}
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
