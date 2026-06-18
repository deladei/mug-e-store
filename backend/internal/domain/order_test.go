package domain

import (
	"errors"
	"testing"
)

// allStatuses is every value the orders.status CHECK permits.
var allStatuses = []Status{
	StatusPendingPayment, StatusPaid, StatusPreparing, StatusReady,
	StatusOutForDelivery, StatusCompleted, StatusCancelled,
}

func TestValidStatus(t *testing.T) {
	for _, s := range allStatuses {
		if !ValidStatus(s) {
			t.Errorf("ValidStatus(%q) = false, want true", s)
		}
	}
	for _, s := range []Status{"", "PAID", "shipped", "pending"} {
		if ValidStatus(s) {
			t.Errorf("ValidStatus(%q) = true, want false", s)
		}
	}
}

func TestValidFulfilment(t *testing.T) {
	for _, f := range []Fulfilment{FulfilmentPickup, FulfilmentDelivery} {
		if !ValidFulfilment(f) {
			t.Errorf("ValidFulfilment(%q) = false, want true", f)
		}
	}
	for _, f := range []Fulfilment{"", "PICKUP", "ship", "dine_in"} {
		if ValidFulfilment(f) {
			t.Errorf("ValidFulfilment(%q) = true, want false", f)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	terminal := map[Status]bool{StatusCompleted: true, StatusCancelled: true}
	for _, s := range allStatuses {
		if got := IsTerminal(s); got != terminal[s] {
			t.Errorf("IsTerminal(%q) = %v, want %v", s, got, terminal[s])
		}
	}
}

// legalEdges is the authoritative set of allowed (from→to) transitions per
// fulfilment type, derived from the API brief (pickup vs delivery paths),
// the TRD §6 (cancellation only early; fulfilment-awareness), and the
// webhook-only pending_payment→paid edge. The implementation is tested
// against this independent declaration.
func legalEdges(f Fulfilment) map[[2]Status]bool {
	common := [][2]Status{
		{StatusPendingPayment, StatusPaid},
		{StatusPendingPayment, StatusCancelled},
		{StatusPaid, StatusPreparing},
		{StatusPaid, StatusCancelled},
		{StatusPreparing, StatusReady},
	}
	var specific [][2]Status
	switch f {
	case FulfilmentPickup:
		specific = [][2]Status{{StatusReady, StatusCompleted}}
	case FulfilmentDelivery:
		specific = [][2]Status{
			{StatusReady, StatusOutForDelivery},
			{StatusOutForDelivery, StatusCompleted},
		}
	}
	out := map[[2]Status]bool{}
	for _, e := range append(common, specific...) {
		out[e] = true
	}
	return out
}

// TestCanTransition_FullMatrix exhaustively checks every (from,to) pair for
// both fulfilment types against legalEdges. This is the highest-value test
// in the codebase: it pins down the entire lifecycle graph and guarantees no
// stray edge can be added unnoticed.
func TestCanTransition_FullMatrix(t *testing.T) {
	for _, f := range []Fulfilment{FulfilmentPickup, FulfilmentDelivery} {
		edges := legalEdges(f)
		for _, from := range allStatuses {
			for _, to := range allStatuses {
				wantLegal := edges[[2]Status{from, to}]
				err := CanTransition(f, from, to)
				if wantLegal && err != nil {
					t.Errorf("%s: CanTransition(%q→%q) = %v, want legal", f, from, to, err)
				}
				if !wantLegal && err == nil {
					t.Errorf("%s: CanTransition(%q→%q) = nil, want rejected", f, from, to)
				}
			}
		}
	}
}

func TestCanTransition_FulfilmentAsymmetry(t *testing.T) {
	// A pickup order can never enter out_for_delivery.
	if err := CanTransition(FulfilmentPickup, StatusReady, StatusOutForDelivery); err == nil {
		t.Error("pickup ready→out_for_delivery should be rejected")
	}
	// A delivery order can never reach completed without shipping first.
	if err := CanTransition(FulfilmentDelivery, StatusReady, StatusCompleted); err == nil {
		t.Error("delivery ready→completed should be rejected")
	}
	// ...but those same edges are legal for the other fulfilment type.
	if err := CanTransition(FulfilmentPickup, StatusReady, StatusCompleted); err != nil {
		t.Errorf("pickup ready→completed should be legal, got %v", err)
	}
	if err := CanTransition(FulfilmentDelivery, StatusReady, StatusOutForDelivery); err != nil {
		t.Errorf("delivery ready→out_for_delivery should be legal, got %v", err)
	}
}

func TestCanTransition_Errors(t *testing.T) {
	tests := []struct {
		name     string
		f        Fulfilment
		from, to Status
		wantErr  error
	}{
		{"unknown fulfilment", "dine_in", StatusPaid, StatusPreparing, ErrUnknownFulfilment},
		{"unknown from status", FulfilmentPickup, "bogus", StatusPaid, ErrUnknownStatus},
		{"unknown to status", FulfilmentPickup, StatusPaid, "bogus", ErrUnknownStatus},
		{"from terminal completed", FulfilmentPickup, StatusCompleted, StatusReady, ErrTerminalState},
		{"from terminal cancelled", FulfilmentPickup, StatusCancelled, StatusPaid, ErrTerminalState},
		{"illegal skip", FulfilmentPickup, StatusPendingPayment, StatusCompleted, ErrIllegalTransition},
		{"cancel too late", FulfilmentPickup, StatusPreparing, StatusCancelled, ErrIllegalTransition},
		{"same status not an edge", FulfilmentPickup, StatusPaid, StatusPaid, ErrIllegalTransition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CanTransition(tt.f, tt.from, tt.to)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CanTransition(%q,%q→%q) = %v, want %v", tt.f, tt.from, tt.to, err, tt.wantErr)
			}
		})
	}
}

func TestIsNoOp(t *testing.T) {
	if !IsNoOp(StatusPaid, StatusPaid) {
		t.Error("IsNoOp(paid,paid) = false, want true (idempotent webhook retry)")
	}
	if IsNoOp(StatusPaid, StatusPreparing) {
		t.Error("IsNoOp(paid,preparing) = true, want false")
	}
}
