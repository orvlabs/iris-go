package iris_test

import (
	"context"
	"fmt"

	iris "github.com/orvlabs/iris-go"
)

func ExampleClient_SendEmail() {
	client := iris.New("iris_test_xxxxxxxx")
	_, err := client.SendEmail(context.Background(), iris.Email{
		Sender:      iris.Address{Email: "factures@boutique.sn"},
		To:          []iris.Address{{Email: "awa@example.sn"}},
		Subject:     "Votre facture",
		HTMLContent: "<p>Votre facture est prête.</p>{{bloc_paiement}}",
		Payment: &iris.Payment{
			AmountXOF:   15000,
			Reference:   "FAC-2026-0042",
			Description: "Facture FAC-2026-0042",
			Methods:     []string{"wave", "orange_money"},
		},
	})
	fmt.Println(err != nil)
}
