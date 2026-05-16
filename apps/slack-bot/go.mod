module github.com/crucible/slack-bot

go 1.23

require (
	github.com/crucible/sdk-go v0.0.0
)

replace (
	github.com/crucible/sdk-go => ../../libs/sdk-go
)
