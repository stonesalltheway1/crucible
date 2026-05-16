module github.com/crucible/verify-go

go 1.23

require (
	github.com/crucible/sdk-go v0.0.0
	github.com/crucible/verifier v0.0.0
)

// The runner re-uses the canonical TestReport schema declared in
// apps/verifier/pkg/testreport. We import it via a local replace so
// the per-language runner cannot drift from the dispatcher's contract.
//
// sdk-go is replaced because testreport transitively imports
// cruciblev1 types — without this replace the build would try to
// fetch the SDK from a non-existent remote.
replace (
	github.com/crucible/sdk-go => ../../libs/sdk-go
	github.com/crucible/verifier => ../../apps/verifier
)
