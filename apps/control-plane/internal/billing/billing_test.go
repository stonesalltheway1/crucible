package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

func mkPR(rubric float64, edited, canary bool) VerifiedPR {
	return VerifiedPR{TenantID: "t", TaskID: "t", RepoID: "r",
		RubricScore: rubric, HumanEdited: edited, CanaryHeld: canary,
		LandedAt: time.Now()}
}

func TestQualifies(t *testing.T) {
	if !mkPR(0.9, false, true).Qualifies() {
		t.Error("happy path should qualify")
	}
	if mkPR(0.8, false, true).Qualifies() {
		t.Error("rubric too low should NOT qualify")
	}
	if mkPR(0.9, true, true).Qualifies() {
		t.Error("human-edited should NOT qualify")
	}
	if mkPR(0.9, false, false).Qualifies() {
		t.Error("canary failed should NOT qualify")
	}
}

func TestProTierBilling(t *testing.T) {
	tc := &TestClient{}
	m := New(tc, nil)
	_, _ = m.Subscribe("t", TierPro, 1, "cus_x", "sub_x")

	// 25 included PRs.
	for i := 0; i < 25; i++ {
		billed, err := m.RecordVerifiedPR(context.Background(), mkPR(0.9, false, true))
		if err != nil || !billed {
			t.Fatalf("expected billed: i=%d err=%v billed=%v", i, err, billed)
		}
	}
	bill, _ := m.CurrentBill("t")
	if bill.OverageUSD != 0 {
		t.Errorf("overage at 25=%v", bill.OverageUSD)
	}
	if bill.BaseUSD != 40 {
		t.Errorf("base=%v want 40", bill.BaseUSD)
	}

	// 5 overage PRs.
	for i := 0; i < 5; i++ {
		_, _ = m.RecordVerifiedPR(context.Background(), mkPR(0.9, false, true))
	}
	bill2, _ := m.CurrentBill("t")
	if bill2.OverageUSD != 5*2.5 {
		t.Errorf("overage=%v want 12.50", bill2.OverageUSD)
	}
}

func TestTeamTierPooled(t *testing.T) {
	tc := &TestClient{}
	m := New(tc, nil)
	_, _ = m.Subscribe("t", TierTeam, 5, "cus", "sub")

	// 5 devs * 80 = 400 included.
	for i := 0; i < 401; i++ {
		_, _ = m.RecordVerifiedPR(context.Background(), mkPR(0.9, false, true))
	}
	bill, _ := m.CurrentBill("t")
	if bill.OverageUSD != 2.0 {
		t.Errorf("overage=%v want 2.00 (one PR over)", bill.OverageUSD)
	}
	if bill.BaseUSD != 5*120 {
		t.Errorf("base=%v want 600", bill.BaseUSD)
	}
}

func TestOutcomeTierMinimum(t *testing.T) {
	tc := &TestClient{}
	m := New(tc, nil)
	_, _ = m.Subscribe("t", TierOutcome, 1, "cus", "sub")
	for i := 0; i < 10; i++ {
		_, _ = m.RecordVerifiedPR(context.Background(), mkPR(0.9, false, true))
	}
	bill, _ := m.CurrentBill("t")
	if bill.TotalUSD != 500 {
		t.Errorf("total=%v want 500 (minimum)", bill.TotalUSD)
	}
	for i := 0; i < 100; i++ {
		_, _ = m.RecordVerifiedPR(context.Background(), mkPR(0.9, false, true))
	}
	bill2, _ := m.CurrentBill("t")
	if bill2.TotalUSD != 110*8 {
		t.Errorf("total=%v want 880 (above minimum)", bill2.TotalUSD)
	}
}

func TestBYOKFlat(t *testing.T) {
	tc := &TestClient{}
	m := New(tc, nil)
	_, _ = m.Subscribe("t", TierBYOK, 4, "cus", "sub")
	for i := 0; i < 10000; i++ {
		_, _ = m.RecordVerifiedPR(context.Background(), mkPR(0.9, false, true))
	}
	bill, _ := m.CurrentBill("t")
	if bill.TotalUSD != 100 {
		t.Errorf("total=%v want 100 (4 devs * $25)", bill.TotalUSD)
	}
}

func TestRejectedPRsNotBilled(t *testing.T) {
	tc := &TestClient{}
	m := New(tc, nil)
	_, _ = m.Subscribe("t", TierPro, 1, "cus", "sub")
	billed, err := m.RecordVerifiedPR(context.Background(), mkPR(0.7, false, true))
	if err != nil || billed {
		t.Errorf("rubric-failed PR was billed")
	}
	billed, _ = m.RecordVerifiedPR(context.Background(), mkPR(0.9, true, true))
	if billed {
		t.Errorf("human-edited PR was billed")
	}
	bill, _ := m.CurrentBill("t")
	if bill.VerifiedPRsCounted != 0 {
		t.Errorf("count=%d want 0", bill.VerifiedPRsCounted)
	}
}

func TestHardCap(t *testing.T) {
	tc := &TestClient{}
	m := New(tc, nil)
	_, _ = m.Subscribe("t", TierPro, 1, "cus", "sub")
	_ = m.SetHardCap("t", 3)
	for i := 0; i < 3; i++ {
		_, err := m.RecordVerifiedPR(context.Background(), mkPR(0.9, false, true))
		if err != nil {
			t.Fatalf("err at i=%d: %v", i, err)
		}
	}
	_, err := m.RecordVerifiedPR(context.Background(), mkPR(0.9, false, true))
	if err != ErrHardCapReached {
		t.Errorf("err=%v want ErrHardCapReached", err)
	}
}

func TestEmitInvoiceResetsPeriod(t *testing.T) {
	tc := &TestClient{}
	m := New(tc, nil)
	_, _ = m.Subscribe("t", TierPro, 1, "cus", "sub")
	for i := 0; i < 30; i++ {
		_, _ = m.RecordVerifiedPR(context.Background(), mkPR(0.9, false, true))
	}
	id, url, err := m.EmitInvoice(context.Background(), "t")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" || url == "" {
		t.Errorf("empty id/url")
	}
	bill, _ := m.CurrentBill("t")
	if bill.VerifiedPRsCounted != 0 {
		t.Errorf("count not reset: %d", bill.VerifiedPRsCounted)
	}
}

func TestRefundIssuedForRejected(t *testing.T) {
	tc := &TestClient{}
	m := New(tc, nil)
	_, _ = m.Subscribe("t", TierPro, 1, "cus", "sub")
	if err := m.IssueRefundForRejected(context.Background(), "t", "ch_123", 2.50); err != nil {
		t.Fatal(err)
	}
	if len(tc.Refunds) != 1 || tc.Refunds[0].Reason != "verifier-rejected" {
		t.Errorf("refund=%v", tc.Refunds)
	}
}

func TestVerifyStripeSignature(t *testing.T) {
	body := []byte(`{"type":"customer.subscription.created"}`)
	secret := "whsec_test"
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(body)
	header := "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	if !verifyStripeSignature(body, header, secret) {
		t.Error("valid signature rejected")
	}
	if verifyStripeSignature(body, "t=1,v1=bad", secret) {
		t.Error("bad signature accepted")
	}
}

func TestHandleWebhookEvent(t *testing.T) {
	tc := &TestClient{}
	m := New(tc, nil)
	body := []byte(`{"type":"customer.subscription.created","data":{"object":{"id":"in_1","customer":"cus_x","subscription":"sub_x","status":"active"}}}`)
	ev, err := m.HandleWebhook(context.Background(), body, "")
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "customer.subscription.created" || ev.StripeCustomerID != "cus_x" {
		t.Errorf("ev=%+v", ev)
	}
}
