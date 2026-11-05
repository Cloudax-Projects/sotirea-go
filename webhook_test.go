package sotirea

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

// The Standard Webhooks specification's own published example, so this agrees with its library
// in every language.
const (
	exampleSecret = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"
	exampleID     = "msg_p5jXN8AQM9LWM0D4loKWxJek"
	exampleSig    = "v1,g0hM9SsE+OTPJTGt/tmIKtSyZlE3uFJELVlNIOLJ1OE="
	exampleBody   = `{"test": 2432232314}`
)

var exampleAt = time.Unix(1614265330, 0)

func exampleHeader(sig string) http.Header {
	h := http.Header{}
	h.Set("webhook-id", exampleID)
	h.Set("webhook-timestamp", "1614265330")
	h.Set("webhook-signature", sig)
	return h
}

func TestTheSpecificationsExampleVerifies(t *testing.T) {
	if _, err := verifyWebhook(exampleSecret, exampleHeader(exampleSig), []byte(exampleBody), exampleAt); err != nil {
		t.Fatal(err)
	}
	// One of several signatures, as while a secret is changed.
	if _, err := verifyWebhook(exampleSecret, exampleHeader("v1,bm90IHRoaXMgb25l "+exampleSig), []byte(exampleBody), exampleAt); err != nil {
		t.Fatalf("one good signature among several: %v", err)
	}
}

func TestAWebhookThatDoesNotCheckOutIsRefused(t *testing.T) {
	for name, c := range map[string]struct {
		header http.Header
		body   string
		at     time.Time
	}{
		"changed body":    {exampleHeader(exampleSig), `{"test": 2432232315}`, exampleAt},
		"other signature": {exampleHeader("v1,bm90IHRoaXMgb25l"), exampleBody, exampleAt},
		"other version":   {exampleHeader("v2," + exampleSig[3:]), exampleBody, exampleAt},
		"too old":         {exampleHeader(exampleSig), exampleBody, exampleAt.Add(6 * time.Minute)},
		"from the future": {exampleHeader(exampleSig), exampleBody, exampleAt.Add(-6 * time.Minute)},
		"no signature":    {exampleHeader(""), exampleBody, exampleAt},
	} {
		if _, err := verifyWebhook(exampleSecret, c.header, []byte(c.body), c.at); !errors.Is(err, ErrWebhookSignature) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAnEventsDataIsReadByItsType(t *testing.T) {
	e := Event{Type: EventPaymentSent, Data: []byte(`{"object":"payment","id":"api:1:order-1","status":"sent","tx_hash":"0xabc"}`)}
	p, err := e.Payment()
	if err != nil || p.ID != "api:1:order-1" || p.TxHash == nil || *p.TxHash != "0xabc" {
		t.Fatalf("read %+v (%v)", p, err)
	}
	if _, err := e.Deposit(); err == nil {
		t.Fatal("a payment event was read as a deposit")
	}
}
