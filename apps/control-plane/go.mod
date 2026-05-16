module github.com/crucible/control-plane

go 1.23

require (
	github.com/anthropics/anthropic-sdk-go v1.43.0
	github.com/crucible/attestation v0.0.0
	github.com/crucible/policy v0.0.0
	github.com/crucible/sdk-go v0.0.0
	github.com/oklog/ulid/v2 v2.1.0
	github.com/openai/openai-go/v3 v3.35.0
	google.golang.org/genai v1.57.0
)

replace (
	github.com/crucible/attestation => ../../libs/attestation
	github.com/crucible/policy => ../../libs/policy
	github.com/crucible/sdk-go => ../../libs/sdk-go
)
