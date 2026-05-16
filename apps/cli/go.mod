module github.com/crucible/cli

go 1.23

require (
	github.com/crucible/sdk-go v0.0.0
	github.com/spf13/cobra v1.8.1
)

replace github.com/crucible/sdk-go => ../../libs/sdk-go
