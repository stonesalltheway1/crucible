# regression / verifier-tier3-timeout-recovery

Past incident: Tier 3 timeout left the verifier daemon in a state
where subsequent tasks queued forever. This case asserts the Tier 2.5
fallback fires cleanly.
