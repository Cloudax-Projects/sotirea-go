# Sotirea Go client

The Go client for the Sotirea API. Its types and methods are generated from the same Go types as
the API itself (`api_gen.go`, never edited by hand), so they cannot drift from what the API
serves. The guide, every operation and every field are on the API's reference page at `/docs`.

```go
import sotirea "sotirea.com/sdk/go"

client := sotirea.New(os.Getenv("SOTIREA_API"), os.Getenv("SOTIREA_API_KEY"))

me, _, err := client.GetMe(ctx)                    // who the key is, and test or live
vaults, err := client.ListVaults(ctx, sotirea.ListVaultsParams{})
payment, meta, err := client.SendPayment(ctx, "order-1001", sotirea.PaymentRequest{
	VaultID: vaults.Items[0].ID, To: "0x...", Amount: "1000000000000000",
})
payment, _, err = client.GetPayment(ctx, payment.ID)
```

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
