package domain

import "errors"

// PesewasPerPoint is the redemption rate. The PRD fixes it at 100 points = GHS 1,
// and GHS 1 is 100 pesewas, so one point is worth exactly one pesewa. Keeping the
// rate as an integer pesewa value means redemption arithmetic never leaves the
// integer-money domain.
const PesewasPerPoint int64 = 1

// Redemption errors. Callers match them with errors.Is to map a rejection onto
// the right response (insufficient points → 409, a malformed request → 400).
var (
	ErrInsufficientPoints = errors.New("domain: insufficient points")
	ErrInvalidRedemption  = errors.New("domain: invalid redemption amount")
)

// Redemption computes how much a checkout discount is worth and how many points
// it consumes, given the points the customer asked to redeem, their current
// ledger balance, and the order subtotal — all in their natural units (points,
// points, pesewas).
//
// The rules, which this function is the single authority for:
//   - A negative request is invalid.
//   - Redeeming zero is a valid no-op (no discount, nothing spent).
//   - You cannot redeem more points than you hold.
//   - Points are worth one pesewa each, and the discount is capped at the
//     subtotal: points reduce what you pay for coffee, never the delivery fee,
//     and the total can never fall below the delivery fee (hence never below 0).
//   - Only the points that actually produce a discount are spent; if the cap
//     bites, the surplus stays on the balance rather than being burned.
//
// It returns the discount in pesewas and the points to debit (a positive number;
// the caller writes it as a negative ledger delta).
func Redemption(requestPoints, balance, subtotalPesewas int64) (discountPesewas, pointsSpent int64, err error) {
	switch {
	case requestPoints < 0 || subtotalPesewas < 0:
		return 0, 0, ErrInvalidRedemption
	case requestPoints == 0:
		return 0, 0, nil
	case requestPoints > balance:
		return 0, 0, ErrInsufficientPoints
	}

	discountPesewas = requestPoints * PesewasPerPoint
	if discountPesewas > subtotalPesewas {
		discountPesewas = subtotalPesewas
	}
	// Spend only the points that bought the discount, so a cap never burns the
	// surplus. With a 1:1 rate this is the discount value back in points.
	pointsSpent = discountPesewas / PesewasPerPoint
	return discountPesewas, pointsSpent, nil
}
