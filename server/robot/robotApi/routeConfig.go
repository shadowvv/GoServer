package robotApi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/drop/GoServer/server/robot/robotRouter"
	"github.com/drop/GoServer/server/robot/robotUtils"
	"github.com/drop/GoServer/server/tool"
)

type directRouteConfig struct {
	Modules map[string][]directRouteConfigItem `yaml:"modules"`
}

type directRouteConfigItem struct {
	Path      string `yaml:"path"`
	MessageID string `yaml:"messageId"`
}

func loadDirectRoutes(path string) ([]directRoute, error) {
	var cfg directRouteConfig
	if err := tool.LoadYaml(path, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.Modules) == 0 {
		return nil, fmt.Errorf("robot api direct routes is empty")
	}

	moduleNames := make([]string, 0, len(cfg.Modules))
	for module := range cfg.Modules {
		moduleNames = append(moduleNames, module)
	}
	sort.Strings(moduleNames)

	seenPaths := make(map[string]string)
	routes := make([]directRoute, 0)
	for _, rawModule := range moduleNames {
		module := robotUtils.NormalizeString(rawModule)
		if module == "" {
			return nil, fmt.Errorf("robot api direct route module is empty")
		}

		items := cfg.Modules[rawModule]
		for i, item := range items {
			routePath := strings.TrimSpace(item.Path)
			if routePath == "" {
				return nil, fmt.Errorf("robot api direct route path is empty in module %q item[%d]", rawModule, i)
			}
			if !strings.HasPrefix(routePath, "/") {
				return nil, fmt.Errorf("robot api direct route path %q in module %q item[%d] must start with /", routePath, rawModule, i)
			}
			if prev, exists := seenPaths[routePath]; exists {
				return nil, fmt.Errorf("duplicated robot api direct route path %q in module %q, already used by module %q", routePath, rawModule, prev)
			}

			messageID, err := robotUtils.ParseMessageID(item.MessageID)
			if err != nil {
				return nil, fmt.Errorf("invalid robot api direct route messageId %q in module %q item[%d]: %w", item.MessageID, rawModule, i, err)
			}
			if err = robotRouter.ValidateRobotOperationBinding(module, messageID); err != nil {
				return nil, fmt.Errorf("invalid robot api direct route binding in module %q item[%d]: %w", rawModule, i, err)
			}

			seenPaths[routePath] = rawModule
			routes = append(routes, directRoute{
				path:      routePath,
				messageID: messageID,
			})
		}
	}
	if len(routes) == 0 {
		return nil, fmt.Errorf("robot api direct routes is empty")
	}
	return routes, nil
}
