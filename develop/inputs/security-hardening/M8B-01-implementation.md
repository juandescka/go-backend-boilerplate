# M8B-01 — HTTP and bootstrap security

Approved by the user M8 master prompt; follows M8A-01 PASS. Preserve working headers, explicit CORS allowlist, validated trusted proxies, bounded JSON readers, configurable server timeouts/drain, opaque sessions, scoped RBAC and local guarded bootstrap.

Fix only audited gaps: validate requested CORS preflight methods/headers, merge Vary on allowed/denied/no-origin decisions, apply no-store to authentication paths before early middleware rejection, tighten new bootstrap input from historical five to requested fifteen characters (valid UTF-8, max1024 bytes retained). Verify both reported typo and actual secret filename ignore safety without reading content or deleting the file. Document HTTPS/HSTS at TLS termination; no CSP that breaks Scalar. Retain all prior migrations and production credentials. Test actual Gin boundaries and bootstrap terminal negatives plus configuration guards.

Verify and obtain separate-context review, then continue M8C.
