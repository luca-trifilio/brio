package canonical

// Settings holds per-request or per-collection knobs.
type Settings struct {
	// EncodeURL toggles URL encoding of query parameters.
	EncodeURL bool
	// TimeoutMS is the client timeout in milliseconds. 0 means no timeout.
	TimeoutMS int
}
