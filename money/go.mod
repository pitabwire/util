module github.com/pitabwire/util/money

go 1.26

require (
	github.com/cockroachdb/apd/v3 v3.2.1
	github.com/pitabwire/util/decimalx v0.0.0
	google.golang.org/genproto v0.0.0-20250324211829-b45e905df463
)

require google.golang.org/protobuf v1.36.5 // indirect

replace github.com/pitabwire/util/decimalx => ../decimalx
