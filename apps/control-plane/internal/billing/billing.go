// Package billing wires Stripe per docs/00-vision/pricing-and-business.md.
//
// Tiers (decided in pricing-and-business.md §"Pricing tiers (v1, public)"):
//
//   - Pro:        $40 / mo,  25 verified PRs, $2.50 overage
//   - Team:       $120 / dev / mo, 80 PRs/dev, $2.00 overage
//   - Outcome:    $8 / verified PR, $500 / mo minimum
//   - BYOK:       $25 / dev / mo flat, no token markup
//   - Enterprise: $50K / yr base + $400 / node / mo, custom SLA
//
// "Verified PR" means: tests pass on real codebase post-promotion,
// verifier rubric_score ≥ 0.85, no human edits before merge, canary
// holds clean. PRs that DON'T meet the bar are not billed (refund-on-
// reject).
//
// In dev / test we run against Stripe TEST mode (sk_test_...). The
// Phase-8 brief is explicit: "Stripe billing in dev uses Stripe test
// mode with the test keys in .env.local (gitignored); production uses
// real keys via Infisical." Production keys are wired in by the
// release coordination at launch — see GUARDRAILS in the Phase 8
// prompt.
package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Tier is one of the published pricing tiers.
type Tier string

const (
	TierPro        Tier = "pro"
	TierTeam       Tier = "team"
	TierOutcome    Tier = "outcome"
	TierBYOK       Tier = "byok"
	TierEnterprise Tier = "enterprise"
)

// PriceCard is the public pricing.
type PriceCard struct {
	Tier               Tier
	BasePerMonth       float64 // dollars
	IncludedVerifiedPRs int    // per dev/month for Team; per tenant/month for Pro
	OveragePerVerifiedPR float64
	PerDevMonth        float64
	PerVerifiedPR      float64 // for Outcome
	MinimumMonthly     float64 // for Outcome
}

// Cards returns the published price cards.
func Cards() map[Tier]PriceCard {
	return map[Tier]PriceCard{
		TierPro: {
			Tier: TierPro, BasePerMonth: 40, IncludedVerifiedPRs: 25,
			OveragePerVerifiedPR: 2.50,
		},
		TierTeam: {
			Tier: TierTeam, PerDevMonth: 120, IncludedVerifiedPRs: 80,
			OveragePerVerifiedPR: 2.00,
		},
		TierOutcome: {
			Tier: TierOutcome, PerVerifiedPR: 8.00, MinimumMonthly: 500,
		},
		TierBYOK: {
			Tier: TierBYOK, PerDevMonth: 25,
		},
		TierEnterprise: {
			Tier: TierEnterprise, BasePerMonth: 50_000.0 / 12, // $50K/yr
		},
	}
}

// VerifiedPR is the meter unit. The verifier emits a VerifierApproval
// → control-plane records a VerifiedPR row → billing meters it.
type VerifiedPR struct {
	TenantID    string
	TaskID      string
	RepoID      string
	DiffLines   int
	RubricScore float64
	HumanEdited bool
	CanaryHeld  bool
	LandedAt    time.Time
}

// Qualifies returns whether this PR counts against billing per the
// strict definition above. PRs that don't qualify are NOT billed —
// the brand promise.
func (v VerifiedPR) Qualifies() bool {
	return v.RubricScore >= 0.85 && !v.HumanEdited && v.CanaryHeld
}

// Subscription is one tenant's tier + meter.
type Subscription struct {
	TenantID            string
	Tier                Tier
	StripeCustomerID    string
	StripeSubID         string
	DevSeats            int
	HardCapVerifiedPRs  int       // 0 = no cap (Outcome)
	StartedAt           time.Time
	NextResetAt         time.Time
	CurrentPeriodCount  int
	CurrentPeriodUSD    float64
}

// Meter aggregates verified-PR counts.
type Meter struct {
	mu      sync.Mutex
	subs    map[string]*Subscription
	prs     map[string][]VerifiedPR // tenant_id → PRs in current period
	cards   map[Tier]PriceCard
	clock   func() time.Time
	stripe  StripeClient
}

// StripeClient is the contract; the production wiring uses the
// official Stripe Go SDK.
type StripeClient interface {
	ReportUsage(ctx context.Context, subItemID string, qty int, ts time.Time) error
	CreateInvoice(ctx context.Context, customerID string, lineItems []InvoiceLine) (invoiceID string, hostedURL string, err error)
	IssueRefund(ctx context.Context, chargeID string, amountUSD float64, reason string) error
	HandleWebhookEvent(rawBody []byte, signature string) (event WebhookEvent, err error)
}

// InvoiceLine is one invoice item.
type InvoiceLine struct {
	Description string
	Quantity    int
	UnitUSD     float64
	AmountUSD   float64
}

// WebhookEvent is a normalised view of relevant Stripe events.
type WebhookEvent struct {
	Type            string
	StripeCustomerID string
	StripeSubID     string
	InvoiceID       string
	Status          string
}

// New returns a Meter.
func New(stripe StripeClient, now func() time.Time) *Meter {
	if now == nil {
		now = time.Now
	}
	return &Meter{
		subs:  map[string]*Subscription{},
		prs:   map[string][]VerifiedPR{},
		cards: Cards(),
		clock: now,
		stripe: stripe,
	}
}

// Subscribe creates or updates a subscription.
func (m *Meter) Subscribe(tenantID string, tier Tier, devSeats int, stripeCustomerID, stripeSubID string) (*Subscription, error) {
	if _, ok := m.cards[tier]; !ok {
		return nil, fmt.Errorf("billing: unknown tier %q", tier)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	sub := &Subscription{
		TenantID: tenantID, Tier: tier, DevSeats: devSeats,
		StripeCustomerID: stripeCustomerID, StripeSubID: stripeSubID,
		StartedAt: m.clock().UTC(),
		NextResetAt: nextMonthBoundary(m.clock().UTC()),
	}
	m.subs[tenantID] = sub
	return sub, nil
}

// RecordVerifiedPR appends a PR to the current period and reports
// usage to Stripe if the PR qualifies.
func (m *Meter) RecordVerifiedPR(ctx context.Context, pr VerifiedPR) (billed bool, err error) {
	if !pr.Qualifies() {
		// Brand promise: rejected/edited PRs are not billed.
		return false, nil
	}
	m.mu.Lock()
	sub, ok := m.subs[pr.TenantID]
	if !ok {
		m.mu.Unlock()
		return false, fmt.Errorf("billing: no subscription for tenant %q", pr.TenantID)
	}
	m.maybeResetPeriodLocked(sub)
	if err := m.checkHardCapLocked(sub); err != nil {
		m.mu.Unlock()
		return false, err
	}
	m.prs[pr.TenantID] = append(m.prs[pr.TenantID], pr)
	sub.CurrentPeriodCount++
	sub.CurrentPeriodUSD = m.computePeriodUSDLocked(sub, m.prs[pr.TenantID])
	subSnapshot := *sub
	m.mu.Unlock()

	if m.stripe == nil {
		return true, nil
	}
	// Report to Stripe meter (subscription_item id encoded in StripeSubID
	// for this Phase-8 commit; production wires a separate item id).
	return true, m.stripe.ReportUsage(ctx, subSnapshot.StripeSubID, 1, pr.LandedAt)
}

// IssueRefundForRejected refunds a PR that was billed but later
// rejected (e.g., canary failed after the period invoice was issued).
func (m *Meter) IssueRefundForRejected(ctx context.Context, tenantID, chargeID string, amountUSD float64) error {
	if m.stripe == nil {
		return errors.New("billing: stripe client not configured")
	}
	return m.stripe.IssueRefund(ctx, chargeID, amountUSD, "verifier-rejected")
}

// HandleWebhook processes a Stripe webhook payload.
func (m *Meter) HandleWebhook(_ context.Context, body []byte, signature string) (WebhookEvent, error) {
	if m.stripe == nil {
		return WebhookEvent{}, errors.New("billing: stripe client not configured")
	}
	return m.stripe.HandleWebhookEvent(body, signature)
}

// CurrentBill returns the current-period bill for the tenant.
func (m *Meter) CurrentBill(tenantID string) (Bill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub, ok := m.subs[tenantID]
	if !ok {
		return Bill{}, fmt.Errorf("billing: no subscription for tenant %q", tenantID)
	}
	return Bill{
		TenantID:           tenantID,
		Tier:               sub.Tier,
		VerifiedPRsCounted: sub.CurrentPeriodCount,
		BaseUSD:            m.baseUSDLocked(sub),
		OverageUSD:         m.overageUSDLocked(sub),
		TotalUSD:           sub.CurrentPeriodUSD,
		PeriodStart:        sub.StartedAt,
		PeriodEnd:          sub.NextResetAt,
	}, nil
}

// Bill is the current-period summary.
type Bill struct {
	TenantID           string    `json:"tenant_id"`
	Tier               Tier      `json:"tier"`
	VerifiedPRsCounted int       `json:"verified_prs_counted"`
	BaseUSD            float64   `json:"base_usd"`
	OverageUSD         float64   `json:"overage_usd"`
	TotalUSD           float64   `json:"total_usd"`
	PeriodStart        time.Time `json:"period_start"`
	PeriodEnd          time.Time `json:"period_end"`
}

// EmitInvoice creates a Stripe invoice for the current period and
// resets the meter.
func (m *Meter) EmitInvoice(ctx context.Context, tenantID string) (string, string, error) {
	m.mu.Lock()
	sub, ok := m.subs[tenantID]
	if !ok {
		m.mu.Unlock()
		return "", "", fmt.Errorf("billing: no subscription for tenant %q", tenantID)
	}
	bill := Bill{
		TenantID: tenantID, Tier: sub.Tier,
		VerifiedPRsCounted: sub.CurrentPeriodCount,
		BaseUSD: m.baseUSDLocked(sub), OverageUSD: m.overageUSDLocked(sub),
		TotalUSD: sub.CurrentPeriodUSD,
	}
	customer := sub.StripeCustomerID
	m.mu.Unlock()

	if m.stripe == nil {
		return "", "", errors.New("billing: stripe client not configured")
	}
	lines := []InvoiceLine{}
	if bill.BaseUSD > 0 {
		lines = append(lines, InvoiceLine{
			Description: "Crucible " + string(sub.Tier) + " — base subscription",
			Quantity:    1, UnitUSD: bill.BaseUSD, AmountUSD: bill.BaseUSD,
		})
	}
	if bill.OverageUSD > 0 {
		lines = append(lines, InvoiceLine{
			Description: fmt.Sprintf("Verified PR overage (%d above plan)", overageCount(sub, m.cards[sub.Tier])),
			Quantity:    overageCount(sub, m.cards[sub.Tier]), UnitUSD: m.cards[sub.Tier].OveragePerVerifiedPR,
			AmountUSD: bill.OverageUSD,
		})
	}
	id, url, err := m.stripe.CreateInvoice(ctx, customer, lines)
	if err != nil {
		return "", "", err
	}
	m.mu.Lock()
	sub.CurrentPeriodCount = 0
	sub.CurrentPeriodUSD = 0
	sub.StartedAt = m.clock().UTC()
	sub.NextResetAt = nextMonthBoundary(sub.StartedAt)
	m.prs[tenantID] = nil
	m.mu.Unlock()
	return id, url, nil
}

// SetHardCap installs a per-period hard cap (admission rejects new
// PRs after the cap).
func (m *Meter) SetHardCap(tenantID string, cap int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub, ok := m.subs[tenantID]
	if !ok {
		return fmt.Errorf("billing: no subscription for tenant %q", tenantID)
	}
	sub.HardCapVerifiedPRs = cap
	return nil
}

// --- Internals ---

func (m *Meter) maybeResetPeriodLocked(sub *Subscription) {
	if !m.clock().UTC().Before(sub.NextResetAt) {
		sub.StartedAt = m.clock().UTC()
		sub.NextResetAt = nextMonthBoundary(sub.StartedAt)
		sub.CurrentPeriodCount = 0
		sub.CurrentPeriodUSD = 0
		m.prs[sub.TenantID] = nil
	}
}

func (m *Meter) checkHardCapLocked(sub *Subscription) error {
	if sub.HardCapVerifiedPRs == 0 {
		return nil
	}
	if sub.CurrentPeriodCount >= sub.HardCapVerifiedPRs {
		return ErrHardCapReached
	}
	return nil
}

func (m *Meter) computePeriodUSDLocked(sub *Subscription, prs []VerifiedPR) float64 {
	base := m.baseUSDLocked(sub)
	overage := m.overageUSDLocked(sub)
	if sub.Tier == TierOutcome {
		card := m.cards[TierOutcome]
		raw := float64(len(prs)) * card.PerVerifiedPR
		if raw < card.MinimumMonthly {
			raw = card.MinimumMonthly
		}
		return raw
	}
	return base + overage
}

func (m *Meter) baseUSDLocked(sub *Subscription) float64 {
	card := m.cards[sub.Tier]
	switch sub.Tier {
	case TierPro:
		return card.BasePerMonth
	case TierTeam:
		return card.PerDevMonth * float64(sub.DevSeats)
	case TierOutcome:
		// Minimum is enforced in computePeriodUSD; base alone is 0.
		return 0
	case TierBYOK:
		return card.PerDevMonth * float64(sub.DevSeats)
	case TierEnterprise:
		return card.BasePerMonth
	}
	return 0
}

func (m *Meter) overageUSDLocked(sub *Subscription) float64 {
	card := m.cards[sub.Tier]
	switch sub.Tier {
	case TierPro:
		over := sub.CurrentPeriodCount - card.IncludedVerifiedPRs
		if over <= 0 {
			return 0
		}
		return float64(over) * card.OveragePerVerifiedPR
	case TierTeam:
		included := card.IncludedVerifiedPRs * sub.DevSeats
		over := sub.CurrentPeriodCount - included
		if over <= 0 {
			return 0
		}
		return float64(over) * card.OveragePerVerifiedPR
	}
	return 0
}

func overageCount(sub *Subscription, card PriceCard) int {
	switch sub.Tier {
	case TierPro:
		over := sub.CurrentPeriodCount - card.IncludedVerifiedPRs
		if over < 0 {
			return 0
		}
		return over
	case TierTeam:
		included := card.IncludedVerifiedPRs * sub.DevSeats
		over := sub.CurrentPeriodCount - included
		if over < 0 {
			return 0
		}
		return over
	}
	return 0
}

func nextMonthBoundary(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
}

// ErrHardCapReached is returned when a PR submission would exceed the
// per-period hard cap.
var ErrHardCapReached = errors.New("billing: per-period hard cap reached")

// MarshalSubscription is a tiny helper for the API surface.
func MarshalSubscription(s *Subscription) []byte {
	body, _ := json.Marshal(s)
	return body
}
