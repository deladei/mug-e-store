package domain

import (
	"errors"
	"testing"
)

func TestRedemption(t *testing.T) {
	tests := []struct {
		name         string
		request      int64
		balance      int64
		subtotal     int64
		wantDiscount int64
		wantSpent    int64
		wantErr      error
	}{
		{name: "zero request is a no-op", request: 0, balance: 500, subtotal: 1000, wantDiscount: 0, wantSpent: 0},
		{name: "redeem within balance and subtotal", request: 300, balance: 500, subtotal: 1000, wantDiscount: 300, wantSpent: 300},
		{name: "redeem exactly the balance", request: 500, balance: 500, subtotal: 1000, wantDiscount: 500, wantSpent: 500},
		{name: "more than balance is rejected", request: 501, balance: 500, subtotal: 1000, wantErr: ErrInsufficientPoints},
		{name: "capped at subtotal, surplus not spent", request: 500, balance: 500, subtotal: 300, wantDiscount: 300, wantSpent: 300},
		{name: "cap leaves the rest on the balance", request: 400, balance: 1000, subtotal: 250, wantDiscount: 250, wantSpent: 250},
		{name: "zero subtotal yields no discount", request: 100, balance: 100, subtotal: 0, wantDiscount: 0, wantSpent: 0},
		{name: "negative request is invalid", request: -1, balance: 500, subtotal: 1000, wantErr: ErrInvalidRedemption},
		{name: "negative subtotal is invalid", request: 100, balance: 500, subtotal: -1, wantErr: ErrInvalidRedemption},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discount, spent, err := Redemption(tt.request, tt.balance, tt.subtotal)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if discount != tt.wantDiscount {
				t.Errorf("discount = %d, want %d", discount, tt.wantDiscount)
			}
			if spent != tt.wantSpent {
				t.Errorf("spent = %d, want %d", spent, tt.wantSpent)
			}
		})
	}
}
