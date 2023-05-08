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
	LoadHeatMap()
	keys := make([]string, len(heatmap))
	i := 0
	for k := range heatmap {
		keys[i] = k
		i++
	}
	fmt.Println(keys)
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
