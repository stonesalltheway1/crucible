module github.com/crucible/promotion-gate

go 1.23

require (
	github.com/crucible/attestation v0.0.0
	github.com/crucible/policy v0.0.0
	github.com/crucible/sdk-go v0.0.0
	github.com/oklog/ulid/v2 v2.1.0
	github.com/open-policy-agent/opa v1.16.2
)

replace (
	github.com/crucible/attestation => ../../libs/attestation
	github.com/crucible/policy => ../../libs/policy
	github.com/crucible/sdk-go => ../../libs/sdk-go
)
