package iris

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
)

func TestSendEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		if r.URL.Path != "/v3/smtp/email" || r.Header.Get("api-key") != "iris_test_k" || got["subject"] != "Votre commande" {
			t.Errorf("unexpected request %s %v %v", r.URL.Path, r.Header, got)
		}
		if _, ok := got["payment"]; ok {
			t.Error("empty payment sent")
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"messageId":"<m1@boutique.sn>"}`))
	}))
	defer srv.Close()
	c := New("iris_test_k")
	c.BaseURL = srv.URL
	sent, err := c.SendEmail(context.Background(), Email{
		Sender: Address{Email: "commandes@boutique.sn"}, To: []Address{{Email: "awa@example.sn"}},
		Subject: "Votre commande", HTMLContent: "<p>Merci</p>",
	})
	if err != nil || sent.MessageID != "<m1@boutique.sn>" {
		t.Fatalf("got %+v, %v", sent, err)
	}
}

func TestSendEmailError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"code":"not_enough_credits","message":"Crédits épuisés"}`))
	}))
	defer srv.Close()
	c := New("iris_k")
	c.BaseURL = srv.URL
	_, err := c.SendEmail(context.Background(), Email{Sender: Address{Email: "a@b.sn"}})
	var e *Error
	if !errors.As(err, &e) || e.Status != 402 || e.Code != "not_enough_credits" || e.Retryable() {
		t.Fatalf("got %v", err)
	}
}

func TestVerifyWebhook(t *testing.T) {
	const secret = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"
	body := []byte(`{"id":"evt_1","type":"email.delivered","createdAt":"2026-11-17T10:00:03Z","data":{"messageId":"<m1@boutique.sn>","email":"awa@example.sn"}}`)
	wh, _ := standardwebhooks.NewWebhook(secret)
	now := time.Now()
	sig, err := wh.Sign("evt_1", now, body)
	if err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set("webhook-id", "evt_1")
	h.Set("webhook-timestamp", strconv.FormatInt(now.Unix(), 10))
	h.Set("webhook-signature", sig)
	ev, err := VerifyWebhook(secret, h, body)
	if err != nil || ev.Type != "email.delivered" || ev.Data.Email != "awa@example.sn" {
		t.Fatalf("got %+v, %v", ev, err)
	}
	if _, err := VerifyWebhook(secret, h, append(body, ' ')); err == nil {
		t.Fatal("tampered body accepted")
	}
}
