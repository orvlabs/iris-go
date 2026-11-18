# iris-go

Go client for [Iris](https://iris.orvlabs.com), transactional email for Senegal.

```sh
go get github.com/orvlabs/iris-go
```

## Send an email

```go
client := iris.New(os.Getenv("IRIS_API_KEY")) // iris_… or iris_test_…

sent, err := client.SendEmail(ctx, iris.Email{
	Sender:      iris.Address{Email: "commandes@boutique.sn", Name: "Boutique"},
	To:          []iris.Address{{Email: "awa@example.sn", Name: "Awa"}},
	Subject:     "Votre commande",
	HTMLContent: "<p>Merci {{params.prenom}} !</p>",
	Params:      map[string]any{"prenom": "Awa"},
})
var apiErr *iris.Error
if errors.As(err, &apiErr) {
	// apiErr.Status, apiErr.Code (not_enough_credits, key_scope…), apiErr.Retryable()
}
```

Keep the key on your server. A test key (`iris_test_…`) is validated like a real one but delivers nothing and uses no units.

## Verify a webhook

```go
func handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	ev, err := iris.VerifyWebhook(os.Getenv("IRIS_WEBHOOK_SECRET"), r.Header, body) // whsec_…
	if err != nil {
		http.Error(w, "bad signature", http.StatusUnauthorized)
		return
	}
	// ev.ID (deduplicate on it), ev.Type (email.delivered, email.bounced…), ev.Data.Email
	w.WriteHeader(http.StatusNoContent)
}
```

Every route, field and error code: [API reference](https://iris.orvlabs.com/docs).

## License

MIT
