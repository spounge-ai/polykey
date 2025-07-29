module github.com/spounge-ai/polykey

go 1.24.5

require (
	github.com/spounge-ai/spounge-proto/gen/go v1.2.2
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.36.6
)

replace github.com/spounge-ai/polykey-service => ./

require (
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
)
