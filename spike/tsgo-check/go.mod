// The module path sits under the upstream module path on purpose: Go checks its
// internal/ rule against import paths, so this is what allows the internal imports.
// This is a separate module so the spike stays out of the root module's ./...
module github.com/microsoft/TypeScript/tsc/rbxtsgo/spike

go 1.27

require github.com/microsoft/TypeScript/tsc v0.0.0

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)

replace github.com/microsoft/TypeScript/tsc => ../../third_party/typescript/tsc
