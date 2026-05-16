package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// TestClient is a deterministic in-memory StripeClient used by dev
// runs and unit tests. Production deployments wire stripe-go.
type TestClient struct {
	mu          sync.Mutex
	WebhookSecret string
	Usage        []TestUsage
	Invoices     []TestInvoice
	Refunds      []TestRefund
}

// TestUsage records a usage report.
type TestUsage struct {
	SubItemID string
	Qty       int
	At        time.Time
}

// TestInvoice records an invoice.
type TestInvoice struct {
	CustomerID string
	Lines      []InvoiceLine
	ID         string
	HostedURL  string
}

// TestRefund records a refund.
type TestRefund struct {
	ChargeID  string
	AmountUSD float64
	Reason    string
}

// ReportUsage records the usage event.
func (c *TestClient) ReportUsage(_ context.Context, subItemID string, qty int, ts time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Usage = append(c.Usage, TestUsage{SubItemID: subItemID, Qty: qty, At: ts})
	return nil
}

// CreateInvoice creates a deterministic invoice.
func (c *TestClient) CreateInvoice(_ context.Context, customerID string, lines []InvoiceLine) (string, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := "inv_test_" + customerID + "_" + time.Now().UTC().Format("20060102150405")
	url := "https://invoices.stripe.test/" + id
	c.Invoices = append(c.Invoices, TestInvoice{CustomerID: customerID, Lines: lines, ID: id, HostedURL: url})
	return id, url, nil
}

// IssueRefund records a refund.
func (c *TestClient) IssueRefund(_ context.Context, chargeID string, amountUSD float64, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Refunds = append(c.Refunds, TestRefund{ChargeID: chargeID, AmountUSD: amountUSD, Reason: reason})
	return nil
}

// HandleWebhookEvent verifies the Stripe signature header and parses the event.
//
// Stripe's signature format: `t=<unix>,v1=<hmac>`. We accept either
// the official format or a literal HMAC for test convenience.
func (c *TestClient) HandleWebhookEvent(body []byte, signature string) (WebhookEvent, error) {
	if c.WebhookSecret != "" {
		if !verifyStripeSignature(body, signature, c.WebhookSecret) {
			return WebhookEvent{}, errors.New("billing: invalid stripe signature")
		}
	}
	var ev struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID         string `json:"id"`
				Customer   string `json:"customer"`
				Subscription string `json:"subscription"`
				Status     string `json:"status"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &ev); err != nil {
		return WebhookEvent{}, err
	}
	return WebhookEvent{
		Type:            ev.Type,
		StripeCustomerID: ev.Data.Object.Customer,
		StripeSubID:     ev.Data.Object.Subscription,
		InvoiceID:       ev.Data.Object.ID,
		Status:          ev.Data.Object.Status,
	}, nil
}

func verifyStripeSignature(body []byte, header, secret string) bool {
	// Format: t=<unix>,v1=<hex>[,v1=<hex>]...
	var ts, sig string
	for _, kv := range splitComma(header) {
		eq := indexEq(kv)
		if eq < 0 {
			continue
		}
		k, v := kv[:eq], kv[eq+1:]
		switch k {
		case "t":
			ts = v
		case "v1":
			if sig == "" {
				sig = v
			}
		}
	}
	if ts == "" || sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(want))
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func indexEq(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return i
		}
	}
	return -1
}
