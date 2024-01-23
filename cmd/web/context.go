package main

// The contextKey type provides unique keys to store and retrieve authentication status without the risk of
// naming collisions.
type contextKey string

const isAuthenticatedContextKey = contextKey("isAuthenticated")
