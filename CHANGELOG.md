# Changelog

## v1.1.0

Everything an integration does in the console, now over the API:

- Networks: `ListNetworks`, each network's coin, finality, recognised tokens and whether
  payments are open there.
- Vaults per customer: `CreateVault` with a `reference`, your own id; the same reference
  answers the same vault. `ListVaults` finds one by reference, and every vault carries it.
- References on payments, `reference` on `PaymentRequest`, answered on every payment and its
  webhooks and filterable; deposits carry their vault's `vault_reference`.
- `QuotePayment`: what sending would come to, with nothing sent: the rules' answer, whether the
  vault has it free, and the fee ceiling in the network's coin, at the price now.
- `CheckAddress`: valid (with its checksum), paid before, approved and from when, a lookalike of
  an address you have paid or approved, and whether it is a contract.
- `SpeedUpPayment` and `CancelPayment` for a stuck payment.
- Webhooks: `ListWebhookEvents`, `ListWebhooks`, `CreateWebhook`, `GetWebhook`,
  `DeleteWebhook`, `EnableWebhook`, `RollWebhookSecret` and `ResendWebhookDelivery`.
- `ListRules`; `ListApprovedAddresses`, `ApproveAddress` and `RemoveApprovedAddress`.
- `RefreshVault` and `GetRefresh`; `ListAudit`.

## v1.0.1

- The import path is `sotirea.com/sdk/go` from this version: `go get sotirea.com/sdk/go` and
  `import sotirea "sotirea.com/sdk/go"`. The package is still `sotirea` and nothing else changed.
  Use v1.0.1 or later; the path v1.0.0 declared never resolved.

## v1.0.0

The first release of the Go client for the Sotirea API, version 1.

- `sotirea.New(baseURL, apiKey, opts...)`, with a method per operation of the API: `GetMe`,
  `ListVaults`, `GetVault`, `ListBalances`, `ListDeposits`, `ListPayments`, `GetPayment`,
  `SendPayment`, `ListGasPools` and `GetStatement`, which answers a period's statement as CSV
  bytes. Their types are written from the API's own, so the client cannot drift from what the
  API serves. Balances, deposits and payments say whether a token is `Recognised`.
- A single object comes with a `Meta` holding its request id; a list answers a `Page[T]` that
  carries its own. A refusal is an `*sotirea.Error` with the API's `Status`, `Code`, `Message`,
  `Rule()`, `RequestID` and `RetryAfter`; `sotirea.IsCode` tests for one.
- `VerifyWebhook` checks a webhook against its signing secret to the Standard Webhooks
  specification, accepting any one of several signatures while a secret is being changed;
  `Event.Payment`, `Event.Deposit` and `Event.GasPool` read an event's data by its type.
- The standard library only, Go 1.22 or later.
