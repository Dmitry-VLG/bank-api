package security

import "testing"

func TestGenerateCardReturnsLuhnValidNumber(t *testing.T) {
	card, err := GenerateCard()
	if err != nil {
		t.Fatal(err)
	}

	if !ValidLuhn(card.Number) {
		t.Fatalf("card number is not luhn-valid: %s", card.Number)
	}

	if len(card.CVV) != 3 {
		t.Fatalf("unexpected cvv length: %d", len(card.CVV))
	}

	if len(card.Last4) != 4 {
		t.Fatalf("unexpected last4: %s", card.Last4)
	}
}
