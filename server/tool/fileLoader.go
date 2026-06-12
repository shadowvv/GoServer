package tool

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadJSON 从JSON文件加载数据
func LoadJSON(filePath string, v interface{}) error {
	if v == nil {
		return errors.New("[system] LoadJSON: target is nil")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("[system] LoadJSON: open file %s: %w", filePath, err)
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {

		}
	}(file)

	dec := json.NewDecoder(file)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("[system] LoadJSON: decode error in %s: %w", filePath, err)
	}
	return nil
}

// LoadYaml 从文件加载YAML数据
func LoadYaml(filePath string, v interface{}) error {
	if v == nil {
		return errors.New("[system] LoadYaml: target is nil")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("[system] LoadYaml: open file %s: %w", filePath, err)
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("[system] LoadYaml: decode error in %s: %w", filePath, err)
	}
	return nil
}
