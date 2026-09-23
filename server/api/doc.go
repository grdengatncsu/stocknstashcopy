// Package api translates HTTP/JSON requests into database operations.
//
// Handler constructors accept a shared database connection and return standard
// net/http handlers, which keeps routing in main.go separate from endpoint
// behavior and makes each endpoint easy to test in isolation.
package api
