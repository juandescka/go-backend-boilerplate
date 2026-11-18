package metrics

import "time"

const (
	RouteUnmatched = "unmatched"

	CacheOperationRead       = "read"
	CacheOperationWrite      = "write"
	CacheOperationInvalidate = "invalidate"
	CacheResultHit           = "hit"
	CacheResultMiss          = "miss"
	CacheResultError         = "error"
	CacheResultInvalid       = "invalid_payload"
	CacheResultSuccess       = "success"

	LoginResultSuccess            = "success"
	LoginResultInvalidCredentials = "invalid_credentials"
	LoginResultRateLimited        = "rate_limited"
	LoginResultError              = "error"

	RefreshResultSuccess     = "success"
	RefreshResultInvalid     = "invalid"
	RefreshResultRateLimited = "rate_limited"
	RefreshResultError       = "error"

	ContextResultValid   = "valid"
	ContextResultInvalid = "invalid"
	ContextResultError   = "error"

	AuthorizationResultAllowed = "allowed"
	AuthorizationResultDenied  = "denied"
	AuthorizationResultError   = "error"

	RateLimitPolicyLoginIP         = "login_ip"
	RateLimitPolicyLoginIdentifier = "login_identifier"
	RateLimitPolicyRefreshIP       = "refresh_ip"

	DatabaseTimeoutQuery       = "query"
	DatabaseTimeoutTransaction = "transaction"
)

// HTTPRecorder owns request lifecycle and panic metric updates.
type HTTPRecorder interface {
	RequestStarted()
	RequestFinished(method, route, status string, duration time.Duration)
	RecordPanic()
}

type CacheRecorder interface {
	RecordUserCacheOperation(operation, result string)
}

type AuthenticationRecorder interface {
	RecordLogin(result string)
	RecordRefresh(result string)
}

type AuthenticationContextRecorder interface {
	RecordAuthenticationContext(result string)
}

type AuthorizationRecorder interface {
	RecordAuthorization(result string)
}

type RateLimitRecorder interface {
	RecordRateLimitBlocked(policy string)
}

type DatabaseRecorder interface {
	RecordDatabaseTimeout(timeoutType string)
}

// NoopRecorder safely satisfies every narrow recorder contract.
type NoopRecorder struct{}

func (NoopRecorder) RequestStarted()                                       {}
func (NoopRecorder) RequestFinished(string, string, string, time.Duration) {}
func (NoopRecorder) RecordPanic()                                          {}
func (NoopRecorder) RecordUserCacheOperation(string, string)               {}
func (NoopRecorder) RecordLogin(string)                                    {}
func (NoopRecorder) RecordRefresh(string)                                  {}
func (NoopRecorder) RecordAuthenticationContext(string)                    {}
func (NoopRecorder) RecordAuthorization(string)                            {}
func (NoopRecorder) RecordRateLimitBlocked(string)                         {}
func (NoopRecorder) RecordDatabaseTimeout(string)                          {}
