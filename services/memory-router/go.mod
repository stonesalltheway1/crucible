module github.com/crucible/memory-router

go 1.22

require (
	github.com/crucible/memory-spec v0.0.0
	github.com/crucible/sdk-go v0.0.0
)

replace (
	github.com/crucible/memory-spec => ../../libs/memory-spec/go
	github.com/crucible/sdk-go => ../../libs/sdk-go
)
