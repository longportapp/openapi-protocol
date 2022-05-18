.PHONY: error_proto control_proto

all: error_proto control_proto

error_proto:
	protoc \
		-I . \
		--go_out=. \
		--go_opt=paths=source_relative \
		./error/error.proto

control_proto:
	protoc \
		-I . \
		--go_out=. \
		--go_opt=paths=source_relative \
		./control/control.proto
