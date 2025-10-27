package fileLoader

import (
	"gopkg.in/yaml.v3"
	"os"
)

func LoadYaml(filePath string, v interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		return err
	}
	return nil
}
