package middleware

import "net/http"

// Secure header names and their default values. Exported so tests (and
// any consumers that want to relax one specific header) can reference
// them by name rather than duplicating string literals.
const (
	HeaderContentTypeOptions        = "X-Content-Type-Options"
	HeaderContentSecurityPolicy     = "Content-Security-Policy"
	HeaderReferrerPolicy            = "Referrer-Policy"
	HeaderStrictTransportSecurity   = "Strict-Transport-Security"
	HeaderCrossOriginResourcePolicy = "Cross-Origin-Resource-Policy"

	ValueContentTypeOptions        = "nosniff"
	ValueContentSecurityPolicy     = "default-src 'none'; frame-ancestors 'none'"
	ValueReferrerPolicy            = "no-referrer"
	ValueStrictTransportSecurity   = "max-age=63072000; includeSubDomains"
	ValueCrossOriginResourcePolicy = "same-origin"
)

// SecureHeaders sets a conservative baseline of HTTP response headers
// appropriate for a JSON API. These defaults are intentionally strict —
// consumers running browser-facing workloads may need to relax CSP,
// COOP/COEP, etc. to match their frontend's requirements.
//
// The headers set here are:
//
//   - X-Content-Type-Options: nosniff
//     Stops browsers from MIME-sniffing responses away from their
//     declared Content-Type, which can otherwise turn a JSON response
//     into an executable HTML page under the right circumstances.
//
//   - Content-Security-Policy: default-src 'none'; frame-ancestors 'none'
//     Because this template serves JSON, the policy forbids loading any
//     resources and disallows being embedded in a frame. HTML APIs
//     should override this.
//
//   - Referrer-Policy: no-referrer
//     Prevents leaking URL path/query to third parties via the Referer
//     header on outbound navigation.
//
//   - Strict-Transport-Security: max-age=63072000; includeSubDomains
//     Tells browsers to use HTTPS for this origin for the next two
//     years. A no-op over plain HTTP, meaningful once a TLS terminator
//     is in front of the service.
//
//   - Cross-Origin-Resource-Policy: same-origin
//     Stops other origins from embedding this response directly via
//     <img>, <script>, etc.
//
// All values are set before the inner handler runs so that handlers
// which write the response body (and thus flush headers) still carry
// them. Handlers that explicitly override a header via w.Header().Set
// after this middleware runs will win, which is the intended override
// path.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set(HeaderContentTypeOptions, ValueContentTypeOptions)
		h.Set(HeaderContentSecurityPolicy, ValueContentSecurityPolicy)
		h.Set(HeaderReferrerPolicy, ValueReferrerPolicy)
		h.Set(HeaderStrictTransportSecurity, ValueStrictTransportSecurity)
		h.Set(HeaderCrossOriginResourcePolicy, ValueCrossOriginResourcePolicy)
		next.ServeHTTP(w, r)
	})
}
