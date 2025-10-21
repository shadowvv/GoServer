package platform

import (
	"google.golang.org/protobuf/proto"
	"log"
)

type Codec struct {
}

func NewCodec() *Codec {
	return &Codec{}
}

func (c *Codec) Marshal(msg proto.Message) ([]byte, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("msg marshal error: %v", err)
		return nil, err
	}
	return data, nil
}

func (c *Codec) Unmarshal(data []byte, msg proto.Message) error {
	if err := proto.Unmarshal(data, msg); err != nil {
		log.Printf("msg unmarshal error: %v", err)
		return err
	}
	return nil
}
