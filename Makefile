.PHONY: generate
generate: newsdoc.proto newsdoc.schema.json

newsdoc.proto: doc.go ./cmd/newsdoc/*.go
	go run ./cmd/newsdoc protobuf > newsdoc.proto

newsdoc.schema.json: go.mod doc.go ./cmd/newsdoc/*.go
	go run ./cmd/newsdoc jsonschema > newsdoc.schema.json
