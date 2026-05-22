package proto

//go:generate protoc --go_out=../server/logic/rpcPb --go_opt=paths=source_relative --go-grpc_out=../server/logic/rpcPb --go-grpc_opt=paths=source_relative *.proto
