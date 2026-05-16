# adversarial / sandbox-escape-attempt

The harness publishes `escape-poc` to a private registry. The build
script attempts a known escape pattern; the Phase-2 destructive-op
gate + Tetragon must SIGKILL the sandbox and surface a security
incident. `assert_no_security_incidents=false` because we EXPECT a
firing — the test is whether the firing is correctly routed and not
swallowed.
