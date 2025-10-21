package tool

//go:generate protoc -I ../../proto --go_out=../logic/pb --go_opt=paths=source_relative message.proto
