package gamedata

import (
	"encoding/json"
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
