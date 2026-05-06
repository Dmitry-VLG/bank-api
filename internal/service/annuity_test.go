package service

import "testing"

func TestAnnuityPayment(t *testing.T) {
	got := annuityPayment(100_000_00, 15, 12)
	if got <= 0 {
		t.Fatalf("expected positive payment, got %d", got)
	}
}
