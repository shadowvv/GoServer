package net

import (
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"google.golang.org/protobuf/proto"
)

func Marshal(pb proto.Message) ([]byte, error) {
	data, err := proto.Marshal(pb)
	if err != nil {
		logger.Error(fmt.Sprintf("pb marshal error: %v", err), 0, 0, 0)
		return nil, err
	}
	return data, nil
}

func Unmarshal(data []byte, msg proto.Message) error {
	if err := proto.Unmarshal(data, msg); err != nil {
		logger.Error(fmt.Sprintf("pb unmarshal error: %v", err), 0, 0, 0)
		return err
	}
	return nil
}
