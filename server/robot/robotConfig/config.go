package robotConfig

import (
	"fmt"
	"sort"
	"strings"

	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/robot/robotRouter"
	"github.com/drop/GoServer/server/robot/robotUtils"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"google.golang.org/protobuf/proto"
)

type RobotConfig struct {
	Logger           logger.LoggerConfig                  `yaml:"loggerConfig"`
	Main             MainConfig                           `yaml:"main"`
	Operations       map[string][]OperationConfigItem     `yaml:"operations"`
	RealOperation    map[string][]RealOperationConfigItem `yaml:"-"`
	CurrentRunModule []RobotRunModule                     `yaml:"-"`
}

type MainConfig struct {
	Login     LoginConfig `yaml:"login"`
	RunConfig RunConfig   `yaml:"runConfig"`
	Modules   []string    `yaml:"modules"`
}

type LoginConfig struct {
	Account  string `yaml:"account"`
	ServerID int32  `yaml:"serverID"`
	LoginURL string `yaml:"loginURL"`
	Channel  int32  `yaml:"channel"`
	Version  string `yaml:"version"`
	Language uint16 `yaml:"language"`
	DeviceID string `yaml:"deviceID"`
	AppID    string `yaml:"appID"`
	Sign     string `yaml:"sign"`
}

type RunConfig struct {
	Count             int    `yaml:"count"`
	Mode              string `yaml:"mode"` // random | custom
	Interval          int64  `yaml:"interval"`
	Duration          int    `yaml:"duration"`
	NamePrefix        string `yaml:"namePrefix"`
	StartupJitterMs   int    `yaml:"startupJitterMs"`
	StartupIntervalMs int    `yaml:"startupIntervalMs"`
}

type OperationConfigItem struct {
	MessageID string                 `yaml:"messageId"`
	Params    map[string]interface{} `yaml:"params"`
}

type RealOperationConfigItem struct {
	MessageID pb.MESSAGE_ID
	Proto     proto.Message
}

type RobotRunModule struct {
	Module     string
	MessageIDs []uint32
}

func (cfg *RobotConfig) BuildRealOperations() error {
	cfg.RealOperation = make(map[string][]RealOperationConfigItem, len(cfg.Operations))

	for module, group := range cfg.Operations {
		seen := make(map[pb.MESSAGE_ID]struct{})
		realGroup := make([]RealOperationConfigItem, 0, len(group))
		for i, item := range group {
			messageID, err := robotUtils.ParseMessageID(item.MessageID)
			if err != nil {
				return fmt.Errorf("invalid operation messageId %q in module %q item[%d]: %w", item.MessageID, module, i, err)
			}
			if _, exists := seen[messageID]; exists {
				return fmt.Errorf("duplicated operation messageId %q in module %q", item.MessageID, module)
			}
			seen[messageID] = struct{}{}

			protoMsg, err := buildOperationProto(module, messageID, item.Params)
			if err != nil {
				return fmt.Errorf("invalid operation params for messageId %q in module %q item[%d]: %w", item.MessageID, module, i, err)
			}

			realGroup = append(realGroup, RealOperationConfigItem{
				MessageID: messageID,
				Proto:     protoMsg,
			})
		}
		cfg.RealOperation[module] = realGroup
	}

	// 构建当前机器人测试的模块
	if err := buildRuntimeModuleMessages(cfg); err != nil {
		return err
	}
	return nil
}

func (c *RobotConfig) FindRealOperation(messageID pb.MESSAGE_ID) (RealOperationConfigItem, bool) {
	if c == nil {
		return RealOperationConfigItem{}, false
	}

	for _, group := range c.RealOperation {
		for _, item := range group {
			if item.MessageID == messageID {
				return item, true
			}
		}
	}
	return RealOperationConfigItem{}, false
}

func (c *RobotConfig) RuntimeNamePrefix() string {
	namePrefix := strings.TrimSpace(c.Main.RunConfig.NamePrefix)
	if namePrefix == "" {
		return "Robot"
	}
	return namePrefix
}

func (c *RobotConfig) RuntimeCount() int {
	if c.Main.RunConfig.Count <= 0 {
		return 1
	}
	return c.Main.RunConfig.Count
}

func (c *RobotConfig) RuntimeStartupJitterMs() int {
	if c.Main.RunConfig.StartupJitterMs <= 0 {
		return 1000
	}
	return c.Main.RunConfig.StartupJitterMs
}

func (c *RobotConfig) RuntimeStartupIntervalMs() int {
	if c.Main.RunConfig.StartupIntervalMs <= 0 {
		return 0
	}
	return c.Main.RunConfig.StartupIntervalMs
}

func LoadConfig(path string) (*RobotConfig, error) {
	var cfg RobotConfig
	if err := tool.LoadYaml(path, &cfg); err != nil {
		return nil, err
	}

	// 检测模块
	normalizedOps, err := normalizeOperationGroups(cfg.Operations)
	if err != nil {
		return nil, err
	}
	cfg.Operations = normalizedOps
	// 检测机器人运行模式
	if err = runtimeModel(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func normalizeOperationGroups(ops map[string][]OperationConfigItem) (map[string][]OperationConfigItem, error) {
	normalizedModules := make(map[string]string)
	out := make(map[string][]OperationConfigItem, len(ops))

	for rawModule, group := range ops {
		module := robotUtils.NormalizeString(rawModule)
		if module == "" {
			return nil, fmt.Errorf("empty module name in operations")
		}
		if prevRaw, exists := normalizedModules[module]; exists && prevRaw != rawModule {
			return nil, fmt.Errorf("duplicated module name after normalization: %q and %q", prevRaw, rawModule)
		}
		normalizedModules[module] = rawModule

		if _, exists := out[module]; exists {
			return nil, fmt.Errorf("duplicated normalized module %q", module)
		}
		out[module] = append([]OperationConfigItem(nil), group...)
	}
	return out, nil
}

func buildOperationProto(module string, messageID pb.MESSAGE_ID, params map[string]interface{}) (proto.Message, error) {
	if err := robotRouter.ValidateRobotOperationBinding(module, messageID); err != nil {
		return nil, err
	}

	if len(params) == 0 {
		return nil, nil
	}

	msg, err := robotRouter.BuildProtoMessageByMessageID(messageID)
	if err != nil {
		return nil, err
	}
	if err := robotUtils.BuildMessageWithParams(msg, params); err != nil {
		return nil, err
	}
	return msg, nil
}

func buildRuntimeModuleMessages(c *RobotConfig) error {
	moduleList := normalizeModuleList(c.Main.Modules, c.RealOperation)
	if len(moduleList) == 0 {
		c.CurrentRunModule = nil
		return nil
	}

	groups := make([]RobotRunModule, 0, len(moduleList))
	for _, module := range moduleList {
		group, ok := c.RealOperation[module]
		if !ok || len(group) == 0 {
			return fmt.Errorf("module %q has no operations or is not configured", module)
		}

		ids := make([]uint32, 0, len(group))
		for _, item := range group {
			ids = append(ids, uint32(item.MessageID))
		}
		if len(ids) == 0 {
			return fmt.Errorf("module %q has no operations or is not configured", module)
		}

		groups = append(groups, RobotRunModule{
			Module:     module,
			MessageIDs: ids,
		})
	}
	c.CurrentRunModule = groups
	return nil
}

func normalizeModuleList(modules []string, operationGroups map[string][]RealOperationConfigItem) []string {
	if len(modules) > 0 {
		out := make([]string, 0, len(modules))
		seen := make(map[string]struct{})
		for _, raw := range modules {
			module := robotUtils.NormalizeString(raw)
			if module == "" {
				continue
			}
			if module == "all" {
				return allConfiguredModules(operationGroups)
			}
			if _, exists := seen[module]; exists {
				continue
			}
			seen[module] = struct{}{}
			out = append(out, module)
		}
		return out
	}

	// convenience fallback: use all configured modules when main.modules is omitted.
	return allConfiguredModules(operationGroups)
}

func allConfiguredModules(operationGroups map[string][]RealOperationConfigItem) []string {
	out := make([]string, 0, len(operationGroups))
	seen := make(map[string]struct{})
	for rawModule := range operationGroups {
		module := robotUtils.NormalizeString(rawModule)
		if module == "" {
			continue
		}
		if _, exists := seen[module]; exists {
			continue
		}
		seen[module] = struct{}{}
		out = append(out, module)
	}
	sort.Strings(out)
	return out
}

func runtimeModel(c *RobotConfig) error {
	model := strings.ToLower(strings.TrimSpace(c.Main.RunConfig.Mode))
	if model == "" {
		model = "custom"
	}
	if model != "random" && model != "custom" {
		return fmt.Errorf("unsupported model %q, only random/custom are allowed", model)
	}
	return nil
}
