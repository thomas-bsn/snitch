package httpapi

import "testing"

func TestValidateWebhookURL(t *testing.T) {
	const token = "AbCdEfGhIjKlMnOpQrStUvWxYz0123456789_-abcdefghijklmnopqrstuvwxyz0123"
	address, hint, ok := validateWebhookURL(" https://Discord.com/api/webhooks/123456789012345678/" + token + " ")
	if !ok || address != "https://discord.com/api/webhooks/123456789012345678/"+token || hint != "5678" {
		t.Fatalf("a Discord webhook is accepted and normalized: %q %q %v", address, hint, ok)
	}
	for _, bad := range []string{
		"http://discord.com/api/webhooks/123456789012345678/" + token,
		"https://evil.example/api/webhooks/123456789012345678/" + token,
		"https://discord.com.evil.example/api/webhooks/123456789012345678/" + token,
		"https://discord.com:8443/api/webhooks/123456789012345678/" + token,
		"https://user:pass@discord.com/api/webhooks/123456789012345678/" + token,
		"https://discord.com/api/webhooks/123456789012345678/" + token + "?wait=true",
		"https://discord.com/api/webhooks/123456789012345678/" + token + "#x",
		"https://discord.com/api/webhooks/123456789012345678/" + token + "/github",
		"https://discord.com/api/webhooks/../../users/@me",
		"https://discord.com/api/webhooks/123456789012345678/%2e%2e%2f" + token,
		"https://127.0.0.1/api/webhooks/123456789012345678/" + token,
		"https://[::1]/api/webhooks/123456789012345678/" + token,
		"",
	} {
		if _, _, ok := validateWebhookURL(bad); ok {
			t.Errorf("refused: %q", bad)
		}
	}
}

func TestWebhookMessageIsRedactedAndGrouped(t *testing.T) {
	got := webhookMessage("https://snitch.example", []string{kindAccountRequest, kindAccountRequest, "job_failed"}, []int{0, 0, 3})
	want := "Nouvelle demande d'accès en attente de validation. <https://snitch.example/admin/users/signups> (×2)\nUn traitement a échoué (3). <https://snitch.example/queue>"
	if got != want {
		t.Fatalf("got %q", got)
	}
}
