# Sotirea Go client

The Go client for the Sotirea API. Its types and methods are generated from the same Go types as
the API itself (`api_gen.go`, never edited by hand), so they cannot drift from what the API
serves. The guide, every operation and every field are on the API's reference page at `/docs`.

```go
import sotirea "sotirea.com/sdk/go"

client := sotirea.New(os.Getenv("SOTIREA_API"), os.Getenv("SOTIREA_API_KEY"))

me, _, err := client.GetMe(ctx) // who the key is, and test or live

// One vault per customer: the same reference always answers the same vault.
vault, _, err := client.CreateVault(ctx, sotirea.VaultRequest{Reference: "cust_1001"})

// What sending would come to, with nothing sent: the rules' answer and the fee ceiling.
pay := sotirea.PaymentRequest{VaultID: vault.ID, To: "0x...", Amount: "1000000000000000", Reference: "payout-42"}
quote, _, err := client.QuotePayment(ctx, pay)
if quote.Outcome == "allowed" && quote.GasPoolCovers {
	// The Idempotency-Key names this payment: sending it again never pays twice.
	payment, meta, err := client.SendPayment(ctx, "payout-42", pay)
	// Follow it with client.GetPayment(ctx, payment.ID), or by webhook.
}
```

`ListNetworks` says which networks your organisation can use, `CheckAddress` what is known about
an address before you pay it, and the rest of the API, webhooks, rules, approved addresses, sweeps,
refreshes and the audit trail included, has a method each. A payment's `Purpose` is `sweep` when one
of your sweeps sent it, and a deposit with `Sweep` true is your own money arriving from one: do not
credit a customer for it.

A single object comes with a `Meta` whose `RequestID` is the one to quote when asking Sotirea about
the request; a list's `Page` carries its own `RequestID`.

A refusal is an `*sotirea.Error` with the API's `Status`, `Code`, `Message`, `Rule()` and
`RequestID`; `sotirea.IsCode(err, sotirea.CodeConflict)` tests for one.

Webhooks:

```go
func hook(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	event, err := sotirea.VerifyWebhook(os.Getenv("SOTIREA_WEBHOOK_SECRET"), r.Header, body)
	if err != nil {
		http.Error(w, "bad signature", http.StatusBadRequest)
		return
	}
	if event.Type == sotirea.EventPaymentConfirmed {
		p, _ := event.Payment()
		// p.ID, p.TxHash ...
	}
	w.WriteHeader(http.StatusNoContent)
}
```

It has no dependencies beyond the standard library.

    go get sotirea.com/sdk/go

MIT licensed; see `LICENSE`.
