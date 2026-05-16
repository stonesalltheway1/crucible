package attestation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Service is the high-level attestation facade: build + sign + publish in one.
// Every Crucible component holds exactly one Service instance, wired with a
// Signer and a Publisher at startup.
type Service struct {
	signer    Signer
	publisher Publisher
}

// NewService composes a Signer + Publisher into a Service.
func NewService(signer Signer, publisher Publisher) (*Service, error) {
	if signer == nil {
		return nil, errors.New("attestation: nil signer")
	}
	if publisher == nil {
		return nil, errors.New("attestation: nil publisher")
	}
	return &Service{signer: signer, publisher: publisher}, nil
}

// Signer returns the underlying signer (e.g. for OIDC subject lookups).
func (s *Service) Signer() Signer { return s.signer }

// Emit builds a Statement around the given predicate, signs it, and publishes it.
// Returns the RekorEntry receipt with UUID, log index, and (for local journal)
// LocalJournalFallback = true.
func (s *Service) Emit(ctx context.Context, predicateType, subjectName string, subjectContent []byte, predicate any) (*cruciblev1.RekorEntry, error) {
	digest := SubjectDigest(subjectContent)
	stmt, err := BuildStatement(predicateType, subjectName, digest, predicate)
	if err != nil {
		return nil, err
	}
	env, err := s.signer.SignStatement(stmt)
	if err != nil {
		return nil, fmt.Errorf("attestation: sign: %w", err)
	}
	return s.publisher.Publish(ctx, env)
}

// EmitJSON is a convenience for callers that already have the predicate as
// a marshaled byte slice (e.g. from a previous SignStatement call).
func (s *Service) EmitJSON(ctx context.Context, predicateType, subjectName string, subjectContent, predicateJSON []byte) (*cruciblev1.RekorEntry, error) {
	return s.Emit(ctx, predicateType, subjectName, subjectContent, json.RawMessage(predicateJSON))
}

// Fetch reads an envelope from the underlying publisher by UUID.
func (s *Service) Fetch(ctx context.Context, uuid string) (*cruciblev1.DsseEnvelope, error) {
	return s.publisher.Fetch(ctx, uuid)
}
