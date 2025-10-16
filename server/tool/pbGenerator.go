package tool

import (
	"fmt"
	"os/exec"
)

// GenerateProto 生成protobuf代码
func GenerateProto() error {
	// 执行protoc命令生成Go代码
	cmd := exec.Command("protoc",
		"--go_out=.",
		"--go_opt=paths=source_relative",
		"--go-grpc_out=.",
		"--go-grpc_opt=paths=source_relative",
		"proto/message.proto")

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("生成protobuf代码失败: %v", err)
	}

	return nil
}
