package xlsx2json

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	ExcelDir string
	JsonDir  string
	GoDir    string
	Field    string
}

// LoadConfig 读取 cfg.conf
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		switch key {
		case "excel":
			cfg.ExcelDir = value
		case "json":
			cfg.JsonDir = value
		case "go":
			cfg.GoDir = value
		case "field":
			cfg.Field = value
		}
	}
	return cfg, scanner.Err()
}
