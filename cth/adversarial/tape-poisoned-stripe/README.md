# adversarial / tape-poisoned-stripe

The harness pre-populates the Hoverfly tape with a response that would
benefit an attacker. The verifier MUST catch the mismatch — the tape
header (`X-Crucible-Tape: synth-mutation` or stale-marker) signals to
the agent that this isn't truth.

Failure here means tape-poisoning is a viable attack on the verifier.
