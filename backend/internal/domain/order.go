// Package domain holds the business rules of the Coffee Mug Shop. It has no
// knowledge of HTTP, SQL, or any external service — it is the single source of
// truth for the order lifecycle. Higher layers (store, httpapi) call into it;
// it calls nothing above itself.
package domain

import "errors"

// Status is one of the seven order lifecycle states. The set is mirrored by the
// CHECK constraint on orders.status; this type guards it in Go.
type Status string

const (
	StatusPendingPayment Status = "pending_payment"
	StatusPaid           Status = "paid"
	StatusPreparing      Status = "preparing"
	StatusReady          Status = "ready"
	StatusOutForDelivery Status = "out_for_delivery"
	StatusCompleted      Status = "completed"
	StatusCancelled      Status = "cancelled"
)

// Fulfilment is how an order is handed to the customer. It decides which
// transitions are legal: pickup orders never enter out_for_delivery, and
// delivery orders never reach completed without shipping first.
type Fulfilment string

const (
	FulfilmentPickup   Fulfilment = "pickup"
	FulfilmentDelivery Fulfilment = "delivery"
)

// Sentinel errors returned by CanTransition. Callers match them with
// errors.Is to map a rejection onto the right HTTP response (e.g. an illegal
// transition becomes 409 Conflict).
var (
	ErrUnknownStatus     = errors.New("domain: unknown order status")
	ErrUnknownFulfilment = errors.New("domain: unknown fulfilment type")
	ErrTerminalState     = errors.New("domain: order is in a terminal state")
	ErrIllegalTransition = errors.New("domain: illegal status transition")
)

// validStatuses is the lookup behind ValidStatus.
var validStatuses = map[Status]bool{
	StatusPendingPayment: true,
	StatusPaid:           true,
	StatusPreparing:      true,
	StatusReady:          true,
	StatusOutForDelivery: true,
	StatusCompleted:      true,
	StatusCancelled:      true,
}

// ValidStatus reports whether s is one of the seven lifecycle states.
func ValidStatus(s Status) bool { return validStatuses[s] }

// ValidFulfilment reports whether f is pickup or delivery.
func ValidFulfilment(f Fulfilment) bool {
	return f == FulfilmentPickup || f == FulfilmentDelivery
}

// IsTerminal reports whether s is an end state with no outgoing transitions.
func IsTerminal(s Status) bool {
	return s == StatusCompleted || s == StatusCancelled
}

// IsNoOp reports whether a transition is the order's current status repeated.
// Re-applying the same status is an idempotent success (this is what makes a
// retried Paystack webhook safe), so callers short-circuit it rather than
// asking CanTransition, which treats from==to as a non-edge.
func IsNoOp(from, to Status) bool { return from == to }

// transitions maps each non-terminal status to the statuses it may move to.
// The graph is fulfilment-aware: the only difference between pickup and
// delivery is what follows `ready`. Edges that are common to both fulfilment
// types live in commonTransitions; the fork is added per fulfilment.
var commonTransitions = map[Status][]Status{
	StatusPendingPayment: {StatusPaid, StatusCancelled},
	StatusPaid:           {StatusPreparing, StatusCancelled},
	StatusPreparing:      {StatusReady},
}

// allowedTargets returns the legal next statuses from `from` for fulfilment f.
func allowedTargets(f Fulfilment, from Status) []Status {
	targets := commonTransitions[from]
	if from == StatusReady {
		switch f {
		case FulfilmentPickup:
			return []Status{StatusCompleted}
		case FulfilmentDelivery:
			return []Status{StatusOutForDelivery}
		}
	}
	if from == StatusOutForDelivery && f == FulfilmentDelivery {
		return []Status{StatusCompleted}
	}
	return targets
}

// CanTransition reports whether moving an order of the given fulfilment type
// from one status to another is legal. It returns nil for a legal transition,
// or a sentinel error explaining the rejection:
//
//   - ErrUnknownFulfilment / ErrUnknownStatus for inputs outside the domain;
//   - ErrTerminalState if `from` is completed or cancelled;
//   - ErrIllegalTransition if the edge is not in the lifecycle graph
//     (including from==to, which is a no-op, not a transition — see IsNoOp).
//
// This function is the single authority for the lifecycle; the database CHECK
// guards the column values, but the legality of moving between them lives here.
func CanTransition(f Fulfilment, from, to Status) error {
	if !ValidFulfilment(f) {
		return ErrUnknownFulfilment
	}
	if !ValidStatus(from) || !ValidStatus(to) {
		return ErrUnknownStatus
	}
	if IsTerminal(from) {
		return ErrTerminalState
	}
	for _, t := range allowedTargets(f, from) {
		if t == to {
			return nil
		}
	}
	return ErrIllegalTransition
}
