module github.com/crucible/verifier

go 1.23

require (
	github.com/crucible/attestation v0.0.0
	github.com/crucible/sdk-go v0.0.0
	github.com/oklog/ulid/v2 v2.1.0
)

replace (
	github.com/crucible/attestation => ../../libs/attestation
	github.com/crucible/sdk-go => ../../libs/sdk-go
)
