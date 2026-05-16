# feature-add / auth-rate-limit

Sliding-window rate limit on /api/auth/login. Verifier runs property
tests on the limiter (no false-rejects under load, never permits >5 in
60s).
