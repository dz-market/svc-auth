package interceptor

import (
	"buf.build/go/protovalidate"
	protovalidatemw "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"google.golang.org/grpc"
)

func Validate(validator protovalidate.Validator) grpc.UnaryServerInterceptor {
	return protovalidatemw.UnaryServerInterceptor(validator)
}
