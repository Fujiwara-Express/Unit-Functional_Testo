package client

import "context"

// FirebaseClient defines the interface for sending PUSH notifications via Firebase.
type FirebaseClient interface {
	SendPush(ctx context.Context, userID, message string) error
}
