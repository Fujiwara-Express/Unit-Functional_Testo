package domain

import "fmt"

// validTransitions defines the allowed status transitions for a Pickup.
var validTransitions = map[Status][]Status{
	StatusScheduled:     {StatusAssigned, StatusCancelled},
	StatusAssigned:      {StatusPickedUp, StatusFailedAttempt},
	StatusFailedAttempt: {StatusScheduled},
}

// Transition attempts to move the Pickup to the given status.
// It returns an error if the transition is not valid from the current status.
func (p *Pickup) Transition(to Status) error {
	allowed, ok := validTransitions[p.Status]
	if !ok {
		return fmt.Errorf("%w: no transitions defined from status %s", ErrInvalidTransition, p.Status)
	}
	for _, s := range allowed {
		if s == to {
			if to == StatusFailedAttempt {
				p.AttemptCount++
			}
			p.Status = to
			return nil
		}
	}
	return fmt.Errorf("%w: from %s to %s", ErrInvalidTransition, p.Status, to)
}
