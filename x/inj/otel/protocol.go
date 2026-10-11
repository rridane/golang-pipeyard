package otel

// The OTLP transports the injectors understand, spelled as
// OTEL_EXPORTER_OTLP_PROTOCOL does. gRPC is the default; HTTP goes through any
// ordinary reverse proxy, which gRPC does not.
const (
	ProtocolGRPC = "grpc"
	ProtocolHTTP = "http/protobuf"
)
