package tool

//go:generate powershell -Command "cd ../../proto; protoc --go_out=../server/logic/pb --go_opt=paths=source_relative *.proto"
