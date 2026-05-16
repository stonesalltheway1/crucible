package attestation

import (
	_ "embed"
	"encoding/json"
	"fmt"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Predicate JSON Schemas embedded so consumers can validate signed payloads
// against the source-of-truth without filesystem access. The .json files live
// in libs/twin-spec/schemas/ and are kept in lockstep with the Go types.

//go:embed schemas/write_attestation_v1.json
var schemaWriteAttestation []byte

//go:embed schemas/migration_attestation_v1.json
var schemaMigrationAttestation []byte

//go:embed schemas/service_call_attestation_v1.json
var schemaServiceCallAttestation []byte

//go:embed schemas/destructive_proposal_v1.json
var schemaDestructiveProposal []byte

//go:embed schemas/destructive_approval_v1.json
var schemaDestructiveApproval []byte

//go:embed schemas/test_report_v1.json
var schemaTestReport []byte

//go:embed schemas/verifier_approval_v1.json
var schemaVerifierApproval []byte

//go:embed schemas/verifier_rejection_v1.json
var schemaVerifierRejection []byte

//go:embed schemas/plan_proposal_v1.json
var schemaPlanProposal []byte

//go:embed schemas/plan_approval_v1.json
var schemaPlanApproval []byte

//go:embed schemas/promotion_bundle_v1.json
var schemaPromotionBundle []byte

//go:embed schemas/promotion_approval_v1.json
var schemaPromotionApproval []byte

//go:embed schemas/promotion_outcome_v1.json
var schemaPromotionOutcome []byte

//go:embed schemas/memory_write_v1.json
var schemaMemoryWrite []byte

// SchemaFor returns the raw JSON Schema bytes for a predicate-type URI.
func SchemaFor(predicateType string) ([]byte, error) {
	switch predicateType {
	case cruciblev1.PredicateWriteAttestation:
		return schemaWriteAttestation, nil
	case cruciblev1.PredicateMigrationAttestation:
		return schemaMigrationAttestation, nil
	case cruciblev1.PredicateServiceCallAttestation:
		return schemaServiceCallAttestation, nil
	case cruciblev1.PredicateDestructiveProposal:
		return schemaDestructiveProposal, nil
	case cruciblev1.PredicateDestructiveApproval:
		return schemaDestructiveApproval, nil
	case cruciblev1.PredicateTestReport:
		return schemaTestReport, nil
	case cruciblev1.PredicateVerifierApproval:
		return schemaVerifierApproval, nil
	case cruciblev1.PredicateVerifierRejection:
		return schemaVerifierRejection, nil
	case cruciblev1.PredicatePlanProposal:
		return schemaPlanProposal, nil
	case cruciblev1.PredicatePlanApproval:
		return schemaPlanApproval, nil
	case cruciblev1.PredicatePromotionBundle:
		return schemaPromotionBundle, nil
	case cruciblev1.PredicatePromotionApproval:
		return schemaPromotionApproval, nil
	case cruciblev1.PredicatePromotionOutcome:
		return schemaPromotionOutcome, nil
	case cruciblev1.PredicateMemoryWrite:
		return schemaMemoryWrite, nil
	default:
		return nil, fmt.Errorf("attestation: no schema registered for %q", predicateType)
	}
}

// AllSchemas returns a map of predicate-type URI -> embedded schema bytes.
// Used by tests to assert the full set of predicate types has a schema.
func AllSchemas() map[string][]byte {
	return map[string][]byte{
		cruciblev1.PredicateWriteAttestation:       schemaWriteAttestation,
		cruciblev1.PredicateMigrationAttestation:   schemaMigrationAttestation,
		cruciblev1.PredicateServiceCallAttestation: schemaServiceCallAttestation,
		cruciblev1.PredicateDestructiveProposal:    schemaDestructiveProposal,
		cruciblev1.PredicateDestructiveApproval:    schemaDestructiveApproval,
		cruciblev1.PredicateTestReport:             schemaTestReport,
		cruciblev1.PredicateVerifierApproval:       schemaVerifierApproval,
		cruciblev1.PredicateVerifierRejection:      schemaVerifierRejection,
		cruciblev1.PredicatePlanProposal:           schemaPlanProposal,
		cruciblev1.PredicatePlanApproval:           schemaPlanApproval,
		cruciblev1.PredicatePromotionBundle:        schemaPromotionBundle,
		cruciblev1.PredicatePromotionApproval:      schemaPromotionApproval,
		cruciblev1.PredicatePromotionOutcome:       schemaPromotionOutcome,
		cruciblev1.PredicateMemoryWrite:            schemaMemoryWrite,
	}
}

// ValidateRequired performs a minimal JSON Schema validation focused on the
// `required` keyword for object-typed schemas. Full draft-2020-12 validation
// would require pulling in github.com/santhosh-tekuri/jsonschema; for Phase 1
// the focused check covers the load-bearing assertion (no missing required
// fields in a predicate payload) without adding a dependency.
//
// Phase 2 will swap this for a full JSON Schema validator once the verifier
// pipeline is wired and we need format/pattern/conditional support.
func ValidateRequired(predicateType string, payload []byte) error {
	schemaBytes, err := SchemaFor(predicateType)
	if err != nil {
		return err
	}
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return fmt.Errorf("attestation: parse embedded schema: %w", err)
	}
	requiredAny, ok := schema["required"].([]any)
	if !ok {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal(payload, &obj); err != nil {
		return fmt.Errorf("attestation: parse payload: %w", err)
	}
	for _, r := range requiredAny {
		key, ok := r.(string)
		if !ok {
			continue
		}
		if _, present := obj[key]; !present {
			return fmt.Errorf("attestation: required field %q missing for %s", key, predicateType)
		}
	}
	return nil
}
