//! 50,000-iteration property test — the brand-promise correctness check.
//!
//! Invariant under test:
//!
//! > For every adversarial command input that contains at least one
//! > destructive operation, `Shim::evaluate_command` MUST return a non-
//! > `Approve` outcome (an [`Outcome::Intercept`],
//! > [`Outcome::AutoApprovedTwinScope`], or [`Outcome::ForwardToPromotion`]).
//! >
//! > Tokenisation errors are also acceptable — fail-closed.
//!
//! The generator emits inputs that **always** contain a destructive
//! pattern, possibly wrapped in adversarial syntax: subshells, command
//! substitutions, backticks, brace groups, environment-variable
//! prefixes, modifier prefixes (`time`, `sudo`, `env`, ...), connector
//! noise (semicolons, `&&`, `||`, pipes, backgrounding `&`), and surrounding
//! benign text.
//!
//! 50_000 iterations are used in CI; locally `PROPTEST_CASES=200_000` can
//! be set for a deeper sweep.

use proptest::collection::vec as pvec;
use proptest::prelude::*;
use twin_runtime_shim::{Outcome, Shim};
use crucible_sandbox_spec::SyscallShimPolicy;

const ITERATIONS: u32 = 50_000;

// Each entry is (semantic_name, command_string, expected_intercept_anywhere).
// We hand-curate a corpus of *known-destructive* command fragments. The
// generator wraps these in adversarial syntax; the assertion is that the
// parser still finds them.
const DESTRUCTIVE_FRAGMENTS: &[&str] = &[
    "rm -rf /tmp/x",
    "rm -fr foo",
    "rm -Rf bar",
    "rm --recursive --force baz",
    "rm foo",
    "find . -delete",
    "find /src -exec rm -f {} +",
    "git push --force",
    "git push -f origin main",
    "git push origin +main",
    "git reset --hard origin/main",
    "git clean -fdx",
    "git branch -D feature/old",
    "git tag -d v0.1",
    "git filter-repo --invert-paths",
    "kubectl delete pod payments-7",
    "kubectl drain node1",
    "kubectl scale deployment/foo --replicas=0",
    "helm uninstall checkout",
    "helm rollback billing 4",
    "terraform destroy",
    "terraform taint aws_db_instance.prod",
    "aws s3 rm s3://prod-bucket/customers",
    "aws ec2 terminate-instances --instance-ids i-abc",
    "aws iam delete-user --user-name admin",
    "gcloud compute instances delete prod-web-1",
    "gcloud sql instances delete prod-db",
    "az vm delete --name prod-1",
    "az group delete --name production",
    "railway down",
    "railway destroy",
    "fly destroy myapp",
    "vercel rm payments-prod --yes",
    "heroku apps:destroy production-api",
    "Remove-Item -Recurse -Force C:\\data",
    "redis-cli FLUSHALL",
    "redis-cli -p 6380 FLUSHDB",
    "redis-cli shutdown",
    "mongorestore --drop prod-backup",
    "psql -c 'DROP TABLE users'",
    "psql -c 'TRUNCATE charges'",
    "psql -c 'DROP DATABASE prod'",
    "mysql -e 'ALTER TABLE foo DROP COLUMN bar'",
    "docker rm -f $(docker ps -q)",
    "docker system prune -af --volumes",
    "dd if=/dev/zero of=/dev/sda bs=1M",
    "shred -uvz /etc/passwd",
    "mkfs.ext4 /dev/nvme0n1",
    "mkfs.btrfs /dev/sdb",
    "systemctl stop nginx",
    "systemctl disable sshd",
    "systemctl poweroff",
    "kill -9 1",
    "pkill -9 systemd",
    "chmod -R 777 /etc",
    "chown -R nobody /usr",
    "iptables -F",
    "nft flush ruleset",
];

fn destructive_fragment() -> impl Strategy<Value = &'static str> {
    proptest::sample::select(DESTRUCTIVE_FRAGMENTS)
}

fn benign_fragment() -> impl Strategy<Value = String> {
    proptest::sample::select(&[
        "ls -la",
        "pwd",
        "echo hello",
        "whoami",
        "uname -a",
        "date",
        "id",
        "env",
        "cat /etc/hostname",
        "df -h",
        "free -m",
        "ps aux",
        "go version",
        "node --version",
        "python --version",
        "cargo --version",
        "make build",
        "npm test",
        "git status",
        "git log --oneline -10",
    ])
    .prop_map(String::from)
}

fn modifier_prefix() -> impl Strategy<Value = &'static str> {
    proptest::sample::select(&[
        "", "time ", "sudo ", "doas ", "env ", "nice ", "ionice ", "stdbuf -i0 -o0 -e0 ",
        "FOO=bar ", "FOO=bar BAZ=qux ", "FOO=$(echo bar) ",
    ])
}

#[derive(Debug, Clone)]
enum Wrap {
    None,
    SubShell,
    CmdSubst,
    Backticks,
    BraceGroup,
}

fn wrap_strategy() -> impl Strategy<Value = Wrap> {
    prop_oneof![
        4 => Just(Wrap::None),
        2 => Just(Wrap::SubShell),
        2 => Just(Wrap::CmdSubst),
        1 => Just(Wrap::Backticks),
        1 => Just(Wrap::BraceGroup),
    ]
}

fn apply_wrap(cmd: &str, wrap: &Wrap) -> String {
    match wrap {
        Wrap::None => cmd.to_string(),
        Wrap::SubShell => format!("({cmd})"),
        Wrap::CmdSubst => format!("echo $({cmd})"),
        Wrap::Backticks => format!("echo `{cmd}`"),
        Wrap::BraceGroup => format!("{{ {cmd}; }}"),
    }
}

#[derive(Debug, Clone)]
enum Connector {
    Semicolon,
    AndAnd,
    OrOr,
    Pipe,
    Newline,
    Background,
}

fn connector_strategy() -> impl Strategy<Value = Connector> {
    prop_oneof![
        Just(Connector::Semicolon),
        Just(Connector::AndAnd),
        Just(Connector::OrOr),
        Just(Connector::Pipe),
        Just(Connector::Newline),
        Just(Connector::Background),
    ]
}

fn connect(a: &str, c: &Connector, b: &str) -> String {
    let sep = match c {
        Connector::Semicolon => "; ",
        Connector::AndAnd => " && ",
        Connector::OrOr => " || ",
        Connector::Pipe => " | ",
        Connector::Newline => "\n",
        Connector::Background => " & ",
    };
    format!("{a}{sep}{b}")
}

fn adversarial_command() -> impl Strategy<Value = String> {
    (
        modifier_prefix(),
        destructive_fragment(),
        wrap_strategy(),
        pvec(benign_fragment(), 0..3),
        pvec(connector_strategy(), 0..3),
    )
        .prop_map(|(prefix, dest, wrap, benigns, connectors)| {
            let inner = apply_wrap(&format!("{prefix}{dest}"), &wrap);
            let mut parts: Vec<String> = Vec::new();
            for b in benigns {
                parts.push(b);
            }
            parts.push(inner);
            if parts.len() == 1 {
                return parts.pop().unwrap();
            }
            let mut connectors = connectors.into_iter();
            let mut acc = parts[0].clone();
            for next in &parts[1..] {
                let conn = connectors
                    .next()
                    .unwrap_or(Connector::Semicolon);
                acc = connect(&acc, &conn, next);
            }
            acc
        })
}

fn shim() -> Shim {
    Shim::build(SyscallShimPolicy::default()).expect("default policy must build")
}

proptest! {
    #![proptest_config(ProptestConfig {
        cases: ITERATIONS,
        max_shrink_iters: 1024,
        ..ProptestConfig::default()
    })]

    /// The headline invariant: 50K adversarial inputs containing a known
    /// destructive fragment must NEVER produce [`Outcome::Approve`].
    /// Tokenisation errors are acceptable — that's also fail-closed.
    #[test]
    fn shim_intercepts_50k_adversarial(input in adversarial_command()) {
        let shim = shim();
        match shim.evaluate_command(&input, "task_property") {
            Ok(Outcome::Approve) => {
                prop_assert!(false, "BYPASS: command was approved: {input:?}");
            }
            Ok(Outcome::Intercept(_) | Outcome::AutoApprovedTwinScope(_) | Outcome::ForwardToPromotion(_)) => {
                // Either way the proposal flow ran — invariant holds.
            }
            Err(_) => {
                // Tokenisation failure is also fail-closed.
            }
        }
    }

    /// Real-scope destructives must NEVER take the auto-approve path,
    /// regardless of input shape — even if a path-dependent fragment was
    /// targeted at /work/scratch (the shim still has to look at the
    /// surrounding context).
    #[test]
    fn real_scope_never_auto_approves(input in adversarial_command()) {
        let shim = shim();
        if let Ok(Outcome::AutoApprovedTwinScope(p)) = shim.evaluate_command(&input, "task_property") {
            prop_assert_eq!(
                p.scope,
                twin_runtime_shim::Scope::Twin,
                "auto-approve fired for non-twin scope: {p:?}"
            );
        }
    }

    /// Benign commands must NEVER trigger the gate. We use a separate,
    /// destructive-free generator to make this assertion meaningful.
    #[test]
    fn benign_commands_approve(input in benign_command()) {
        let shim = shim();
        match shim.evaluate_command(&input, "task_property") {
            Ok(Outcome::Approve) => {}
            Ok(other) => prop_assert!(false, "false positive on benign input {input:?}: {other:?}"),
            // Tokenisation failure is acceptable for malformed inputs; but
            // our generator never emits malformed inputs.
            Err(e) => prop_assert!(false, "unexpected tokenisation error on benign input {input:?}: {e}"),
        }
    }
}

fn benign_command() -> impl Strategy<Value = String> {
    (
        pvec(benign_fragment(), 1..4),
        pvec(connector_strategy(), 0..3),
    )
        .prop_map(|(benigns, connectors)| {
            let mut connectors = connectors.into_iter();
            let mut acc = benigns[0].clone();
            for next in &benigns[1..] {
                let conn = connectors
                    .next()
                    .unwrap_or(Connector::Semicolon);
                acc = connect(&acc, &conn, next);
            }
            acc
        })
}
