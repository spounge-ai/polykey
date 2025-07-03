package polykey

//go:generate mkdir -p pb
//go:generate protoc --proto_path=../../proto --go_out=./pb --go_opt=module=github.com/SpoungeAI/polykey-service/pkg/polykey --go-grpc_out=./pb --go-grpc_opt=module=github.com/SpoungeAI/polykey-service/pkg/polykey ../../proto/polykey.proto
