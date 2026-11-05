package sotirea

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Webhooks are signed to the Standard Webhooks specification (standardwebhooks.com): this checks
// one as that specification says, so its library for any language would agree.

// WebhookTolerance is how far a webhook's timestamp may be from now before it is refused, so a
// recorded webhook cannot be replayed later.
const WebhookTolerance = 5 * time.Minute

// ErrWebhookSignature is a webhook whose signature, headers or timestamp do not check out. Answer
// it with a 4xx and do not act on it.
var ErrWebhookSignature = errors.New("sotirea: the webhook's signature does not match")

// VerifyWebhook checks a webhook against its endpoint's secret (whsec_..., shown once when the
// endpoint was added) and answers the event it carries. body must be the request body exactly as
// received, before any parsing.
func VerifyWebhook(secret string, header http.Header, body []byte) (Event, error) {
	return verifyWebhook(secret, header, body, time.Now())
}

func verifyWebhook(secret string, header http.Header, body []byte, now time.Time) (Event, error) {
	id, ts, sigs := header.Get("webhook-id"), header.Get("webhook-timestamp"), header.Get("webhook-signature")
	if id == "" || ts == "" || sigs == "" {
		return Event{}, fmt.Errorf("%w: a webhook header is missing", ErrWebhookSignature)
	}
	seconds, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return Event{}, fmt.Errorf("%w: the timestamp is not a number", ErrWebhookSignature)
	}
	if at := time.Unix(seconds, 0); at.Before(now.Add(-WebhookTolerance)) || at.After(now.Add(WebhookTolerance)) {
		return Event{}, fmt.Errorf("%w: the timestamp is too far from now", ErrWebhookSignature)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil || len(key) == 0 {
		return Event{}, errors.New("sotirea: the webhook secret is not one Sotirea gave (whsec_...)")
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + ts + "."))
	mac.Write(body)
	want := mac.Sum(nil)
	// The header may carry more than one signature, space separated, while a secret is changed.
	for _, sig := range strings.Fields(sigs) {
		version, value, ok := strings.Cut(sig, ",")
		if !ok || version != "v1" {
			continue
		}
		got, err := base64.StdEncoding.DecodeString(value)
		if err == nil && hmac.Equal(got, want) {
			var e Event
			if err := json.Unmarshal(body, &e); err != nil {
				return Event{}, fmt.Errorf("sotirea: a signed webhook that is not an event: %w", err)
			}
			return e, nil
		}
	}
	return Event{}, ErrWebhookSignature
}

// Payment is a payment event's data: for payment.awaiting_approval, .sent, .confirmed, .failed
// and .stuck.
func (e Event) Payment() (PaymentEvent, error) { return eventData[PaymentEvent](e, "payment.") }

// Deposit is a deposit event's data: for deposit.arriving and deposit.received.
func (e Event) Deposit() (DepositEvent, error) { return eventData[DepositEvent](e, "deposit.") }

// GasPool is a gas event's data: for gas.low.
func (e Event) GasPool() (GasPoolEvent, error) { return eventData[GasPoolEvent](e, "gas.") }

func eventData[T any](e Event, prefix string) (T, error) {
	var out T
	if !strings.HasPrefix(e.Type, prefix) {
		return out, fmt.Errorf("sotirea: a %s event does not carry %s data", e.Type, strings.TrimSuffix(prefix, "."))
	}
	if err := json.Unmarshal(e.Data, &out); err != nil {
		return out, fmt.Errorf("sotirea: reading a %s event: %w", e.Type, err)
	}
	return out, nil
}
