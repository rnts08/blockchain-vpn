package blockchain

import "testing"

func TestVerifyPaymentInput(t *testing.T) {
	// Should error if amount < advertised
	err := VerifyPaymentInput(1000, 500)
	if err == nil {
		t.Error("expected error for amount less than price")
	}
	// Should be ok if amount >= advertised
	err = VerifyPaymentInput(1000, 1000)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = VerifyPaymentInput(1000, 1500)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetPaymentVerification(t *testing.T) {
	amount, err := GetPaymentVerification(1000, 1000)
	if err != nil || amount != 1000 {
		t.Errorf("expected (1000, nil), got (%d, %v)", amount, err)
	}
	amount, err = GetPaymentVerification(1000, 1200)
	if err != nil || amount != 1200 {
		t.Errorf("expected (1200, nil), got (%d, %v)", amount, err)
	}
	_, err = GetPaymentVerification(1000, 500)
	if err == nil {
		t.Error("expected error for insufficient amount")
	}
}
