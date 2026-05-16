//! Layer 1 — command-line parse + corpus match.
//!
//! This module accepts a raw `twin.shell.exec(cmd)` string and decomposes it
//! into the set of invoked commands (a single string can invoke many via
//! `;`, `&&`, `||`, `|`, command substitution `$(...)`, backticks, subshell
//! `(...)`). Each extracted command is matched against [`crate::corpus`].
//!
//! Correctness invariants:
//!
//! 1. **Fail-closed tokenisation.** Any unparseable input ([`ParseError`])
//!    is rejected by [`crate::Shim::evaluate_command`], not executed. An
//!    attacker cannot smuggle a destructive op past the gate by emitting
//!    a malformed but syntactically suggestive command.
//!
//! 2. **Recursive descent into substitutions.** A `$(rm -rf /)` inside an
//!    otherwise-benign command produces a destructive match for `rm`.
//!
//! 3. **Connector blindness.** `;`, `&&`, `||`, `|`, `&`, and `\n` all
//!    yield independent command tokens. So do subshells `( ... )` and
//!    bracelists `{ ... ; }`.

use crate::corpus::{DestructivePattern, CORPUS};
use std::fmt;

/// A single command invocation: `argv[0]` is the binary, the rest is argv.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Command {
    /// argv as the user typed it (post-tokenisation, pre-glob-expansion).
    pub argv: Vec<String>,
    /// Optional offset within the original input where this command starts.
    /// Used for richer attestation context; not security-bearing.
    pub source_offset: usize,
}

impl Command {
    /// Binary name (`argv[0]`), or `""` if empty.
    #[must_use]
    pub fn binary(&self) -> &str {
        self.argv.first().map(String::as_str).unwrap_or("")
    }

    /// Returns the argv tail (`argv[1..]`).
    #[must_use]
    pub fn args(&self) -> &[String] {
        self.argv.get(1..).unwrap_or(&[])
    }
}

/// Parse error — surfaced as [`crate::Error::TokenisationFailed`].
#[derive(Debug, Clone)]
pub enum ParseError {
    /// Quoting was unbalanced (`'`, `"`, backtick).
    UnbalancedQuote(char),
    /// `$(...)` was unbalanced.
    UnbalancedCmdSubst,
    /// Subshell `(...)` was unbalanced.
    UnbalancedSubshell,
    /// `{ ... ; }` was unbalanced.
    UnbalancedBraceGroup,
    /// `shell-words` failed to tokenise a segment.
    TokeniserFailure(String),
    /// Recursion limit hit — `$($($($...$)))` nesting protection.
    RecursionLimit,
    /// Empty input.
    Empty,
}

impl fmt::Display for ParseError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::UnbalancedQuote(c) => write!(f, "unbalanced quote {c:?}"),
            Self::UnbalancedCmdSubst => write!(f, "unbalanced $(...) command substitution"),
            Self::UnbalancedSubshell => write!(f, "unbalanced (...) subshell"),
            Self::UnbalancedBraceGroup => write!(f, "unbalanced {{...}} brace group"),
            Self::TokeniserFailure(s) => write!(f, "tokeniser: {s}"),
            Self::RecursionLimit => write!(f, "nesting too deep (substitution recursion limit)"),
            Self::Empty => write!(f, "empty command"),
        }
    }
}

impl std::error::Error for ParseError {}

/// Parsed result: all extracted commands, in source order.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ParsedInput {
    /// Every extracted command, in source order. May be empty for a
    /// whitespace-only input (which the gate treats as a no-op).
    pub commands: Vec<Command>,
}

/// Match outcome.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum MatchResult {
    /// No command in the input matched the destructive corpus.
    Benign,
    /// One or more commands matched. The first match is reported; multiple
    /// matches still produce a single proposal (the agent must escalate
    /// one at a time, by design).
    Destructive(CorpusHit),
}

/// Corpus hit — the matched pattern plus the offending [`Command`].
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CorpusHit {
    /// Pattern semantic id.
    pub pattern_id: &'static str,
    /// Pattern reason text.
    pub reason: &'static str,
    /// The command that matched.
    pub command: Command,
    /// Default scope from the corpus entry.
    pub default_scope: crate::corpus::PatternScope,
    /// Reversibility hint from the corpus entry.
    pub reversibility: crate::corpus::Reversibility,
}

/// The Layer-1 parser.
pub struct Parser {
    corpus: &'static [DestructivePattern],
}

impl Parser {
    /// Build a parser using the global [`CORPUS`].
    #[must_use]
    pub fn with_default_corpus() -> Self {
        Self { corpus: &CORPUS }
    }

    /// Parse `input` into its constituent commands.
    ///
    /// # Errors
    /// Returns [`ParseError`] for malformed input. Fail-closed: the caller
    /// rejects the exec rather than executing a partially-parsed command.
    pub fn parse(&self, input: &str) -> Result<ParsedInput, ParseError> {
        let mut commands = Vec::new();
        extract_commands(input, 0, 0, &mut commands)?;
        Ok(ParsedInput { commands })
    }

    /// Match a parsed input against the corpus. Returns the **first**
    /// destructive hit; ordering follows [`ParsedInput::commands`].
    #[must_use]
    pub fn match_corpus(&self, parsed: &ParsedInput) -> MatchResult {
        for cmd in &parsed.commands {
            if let Some(hit) = self.match_one(cmd) {
                return MatchResult::Destructive(hit);
            }
        }
        MatchResult::Benign
    }

    fn match_one(&self, cmd: &Command) -> Option<CorpusHit> {
        let binary = cmd.binary();
        if binary.is_empty() {
            return None;
        }
        let base = bin_basename(binary);
        for pat in self.corpus {
            if pat.binary == base || pat.extra_binaries.contains(&base) {
                if (pat.arg_predicate)(cmd.args()) {
                    return Some(CorpusHit {
                        pattern_id: pat.id,
                        reason: pat.reason,
                        command: cmd.clone(),
                        default_scope: pat.default_scope,
                        reversibility: pat.reversibility,
                    });
                }
            }
        }
        None
    }
}

/// Strip the leading path on a binary name: `/usr/bin/rm` → `rm`. Windows
/// paths are also stripped (`C:\Windows\System32\rm.exe` → `rm.exe` → `rm`).
fn bin_basename(s: &str) -> &str {
    let last_slash = s.rfind(|c: char| c == '/' || c == '\\');
    let stripped = match last_slash {
        Some(idx) => &s[idx + 1..],
        None => s,
    };
    // Trim `.exe` if present.
    stripped.strip_suffix(".exe").unwrap_or(stripped)
}

// ─────────────────────────────────────────────────────────────────────────────
// Recursive extractor
// ─────────────────────────────────────────────────────────────────────────────

const MAX_RECURSION: u32 = 16;

fn extract_commands(
    input: &str,
    base_offset: usize,
    depth: u32,
    out: &mut Vec<Command>,
) -> Result<(), ParseError> {
    if depth > MAX_RECURSION {
        return Err(ParseError::RecursionLimit);
    }

    // Walk the input, splitting on top-level connectors and recursing into
    // substitutions / subshells. Inside quotes we ignore connectors.
    let mut segments: Vec<(usize, String)> = Vec::new();
    let mut buf = String::new();
    let mut seg_start = base_offset;
    let mut chars = input.char_indices().peekable();
    let mut quote: Option<char> = None;
    let mut paren_depth: u32 = 0;
    let mut brace_depth: u32 = 0;

    while let Some((i, c)) = chars.next() {
        // Inside a quote, accept everything except the closer.
        if let Some(q) = quote {
            buf.push(c);
            if c == q && !preceded_by_backslash(&buf) {
                quote = None;
            }
            continue;
        }

        match c {
            '\\' => {
                buf.push(c);
                if let Some((_, next)) = chars.next() {
                    buf.push(next);
                }
            }
            '\'' | '"' => {
                quote = Some(c);
                buf.push(c);
            }
            '`' => {
                // Backtick command substitution: extract content up to the
                // matching backtick (no nesting in POSIX backticks; we treat
                // the simple case).
                let start = i + c.len_utf8();
                let mut end_idx = None;
                for (j, cc) in input[start..].char_indices() {
                    if cc == '`' && !preceded_by_backslash(&input[start..start + j]) {
                        end_idx = Some(j);
                        break;
                    }
                }
                let end = end_idx.ok_or(ParseError::UnbalancedQuote('`'))?;
                let inner = &input[start..start + end];
                extract_commands(inner, base_offset + start, depth + 1, out)?;
                // Skip past the closing backtick.
                buf.push_str(&format!("`{inner}`"));
                while let Some(&(j, _)) = chars.peek() {
                    if j > start + end {
                        break;
                    }
                    chars.next();
                }
            }
            '$' if peek_is(&chars, '(') => {
                // $( ... ) command substitution. Capture nesting level.
                chars.next(); // consume the '('
                let start = i + 2;
                let mut local_paren = 1;
                let mut end_idx = None;
                let mut local_quote: Option<char> = None;
                for (j, cc) in input[start..].char_indices() {
                    if let Some(q) = local_quote {
                        if cc == q && !preceded_by_backslash(&input[start..start + j]) {
                            local_quote = None;
                        }
                        continue;
                    }
                    match cc {
                        '\'' | '"' => local_quote = Some(cc),
                        '(' => local_paren += 1,
                        ')' => {
                            local_paren -= 1;
                            if local_paren == 0 {
                                end_idx = Some(j);
                                break;
                            }
                        }
                        _ => {}
                    }
                }
                let end = end_idx.ok_or(ParseError::UnbalancedCmdSubst)?;
                let inner = &input[start..start + end];
                extract_commands(inner, base_offset + start, depth + 1, out)?;
                buf.push_str(&format!("$({inner})"));
                // Advance the outer iterator past the inner segment + ')'.
                while let Some(&(j, _)) = chars.peek() {
                    if j > start + end {
                        break;
                    }
                    chars.next();
                }
            }
            '(' if brace_depth == 0 => {
                paren_depth += 1;
                buf.push(c);
            }
            ')' if brace_depth == 0 => {
                if paren_depth == 0 {
                    // Unmatched ')' is a real shell error; fail-closed.
                    return Err(ParseError::UnbalancedSubshell);
                }
                paren_depth -= 1;
                buf.push(c);
                if paren_depth == 0 {
                    // The buffer now contains a `( ... )` subshell — recurse.
                    let inner = strip_outer_parens(&buf);
                    if !inner.is_empty() {
                        extract_commands(inner, seg_start, depth + 1, out)?;
                    }
                    buf.clear();
                    seg_start = base_offset + i + c.len_utf8();
                }
            }
            '{' if paren_depth == 0 => {
                brace_depth += 1;
                buf.push(c);
            }
            '}' if paren_depth == 0 => {
                if brace_depth == 0 {
                    return Err(ParseError::UnbalancedBraceGroup);
                }
                brace_depth -= 1;
                buf.push(c);
                if brace_depth == 0 {
                    let inner = strip_outer_braces(&buf);
                    if !inner.is_empty() {
                        extract_commands(inner, seg_start, depth + 1, out)?;
                    }
                    buf.clear();
                    seg_start = base_offset + i + c.len_utf8();
                }
            }
            ';' | '\n' if paren_depth == 0 && brace_depth == 0 => {
                push_segment(&mut segments, seg_start, std::mem::take(&mut buf));
                seg_start = base_offset + i + c.len_utf8();
            }
            '&' if paren_depth == 0 && brace_depth == 0 && peek_is(&chars, '&') => {
                chars.next();
                push_segment(&mut segments, seg_start, std::mem::take(&mut buf));
                seg_start = base_offset + i + 2;
            }
            '|' if paren_depth == 0 && brace_depth == 0 => {
                if peek_is(&chars, '|') {
                    chars.next();
                    push_segment(&mut segments, seg_start, std::mem::take(&mut buf));
                    seg_start = base_offset + i + 2;
                } else {
                    push_segment(&mut segments, seg_start, std::mem::take(&mut buf));
                    seg_start = base_offset + i + c.len_utf8();
                }
            }
            '&' if paren_depth == 0 && brace_depth == 0 => {
                // Single & = backgrounding. Treat as a connector.
                push_segment(&mut segments, seg_start, std::mem::take(&mut buf));
                seg_start = base_offset + i + c.len_utf8();
            }
            _ => buf.push(c),
        }
    }

    if quote.is_some() {
        return Err(ParseError::UnbalancedQuote(quote.unwrap()));
    }
    if paren_depth != 0 {
        return Err(ParseError::UnbalancedSubshell);
    }
    if brace_depth != 0 {
        return Err(ParseError::UnbalancedBraceGroup);
    }
    push_segment(&mut segments, seg_start, buf);

    for (offset, seg) in segments {
        let trimmed = seg.trim();
        if trimmed.is_empty() {
            continue;
        }
        // Strip leading env-var assignments: `FOO=bar baz` → `baz`. The env
        // assignment is itself not destructive; we want the matcher to see
        // the binary that runs.
        let trimmed = strip_env_assignments(trimmed);
        // Strip common time/sudo/etc prefixes that don't change the matched
        // binary's destructiveness.
        let trimmed = strip_modifier_prefixes(trimmed);
        if trimmed.is_empty() {
            continue;
        }
        let argv = shell_words::split(trimmed)
            .map_err(|e| ParseError::TokeniserFailure(e.to_string()))?;
        if !argv.is_empty() {
            out.push(Command {
                argv,
                source_offset: offset,
            });
        }
    }
    Ok(())
}

fn push_segment(segments: &mut Vec<(usize, String)>, offset: usize, s: String) {
    if !s.trim().is_empty() {
        segments.push((offset, s));
    }
}

fn peek_is(chars: &std::iter::Peekable<std::str::CharIndices<'_>>, target: char) -> bool {
    chars.clone().peek().map(|&(_, c)| c == target).unwrap_or(false)
}

fn preceded_by_backslash(s: &str) -> bool {
    // The buffer ends with `\X` where X is the candidate quote char we just
    // pushed; count the backslashes before X to decide if X is escaped.
    let mut count = 0;
    // We strip the trailing character because the caller has just pushed it.
    let body = &s[..s.len().saturating_sub(1)];
    for c in body.chars().rev() {
        if c == '\\' {
            count += 1;
        } else {
            break;
        }
    }
    count % 2 == 1
}

fn strip_outer_parens(buf: &str) -> &str {
    let trimmed = buf.trim();
    trimmed
        .strip_prefix('(')
        .and_then(|s| s.strip_suffix(')'))
        .map_or(trimmed, str::trim)
}

fn strip_outer_braces(buf: &str) -> &str {
    let trimmed = buf.trim();
    trimmed
        .strip_prefix('{')
        .and_then(|s| s.strip_suffix('}'))
        .map_or(trimmed, str::trim)
        .trim_end_matches(';')
        .trim()
}

/// Strip leading `FOO=bar BAZ=qux` env assignments from a segment.
fn strip_env_assignments(segment: &str) -> &str {
    let mut start = 0;
    for word in segment.split_whitespace() {
        if is_env_assignment(word) {
            start += word.len();
            // Plus the whitespace after.
            while start < segment.len()
                && segment.as_bytes()[start].is_ascii_whitespace()
            {
                start += 1;
            }
        } else {
            break;
        }
    }
    segment[start..].trim_start()
}

fn is_env_assignment(word: &str) -> bool {
    if let Some(eq) = word.find('=') {
        let lhs = &word[..eq];
        !lhs.is_empty()
            && lhs.chars().all(|c| c.is_alphanumeric() || c == '_')
            && !lhs.starts_with(|c: char| c.is_ascii_digit())
    } else {
        false
    }
}

/// Strip wrappers that don't change the destructiveness of what follows.
/// `time rm -rf foo` is just as destructive as `rm -rf foo`; `sudo terraform
/// destroy` is just as destructive too.
fn strip_modifier_prefixes(segment: &str) -> &str {
    const MODIFIERS: &[&str] = &[
        "time",
        "nohup",
        "exec",
        "command",
        "builtin",
        "sudo",
        "doas",
        "su",
        "nice",
        "ionice",
        "stdbuf",
        "unbuffer",
        "env",
    ];
    let mut s = segment.trim_start();
    loop {
        let advanced = MODIFIERS.iter().find_map(|m| {
            let with_space = format!("{m} ");
            s.strip_prefix(&with_space).map(str::trim_start)
        });
        match advanced {
            Some(next) => s = next,
            None => break,
        }
    }
    s
}

#[cfg(test)]
mod tests {
    use super::*;

    fn parse(input: &str) -> Result<ParsedInput, ParseError> {
        Parser::with_default_corpus().parse(input)
    }

    fn extract(input: &str) -> Vec<Vec<&str>> {
        // Helper that returns the argv of every extracted command.
        let result = parse(input).unwrap();
        result
            .commands
            .iter()
            .map(|c| c.argv.iter().map(String::as_str).collect())
            .collect()
    }

    #[test]
    fn simple_command() {
        assert_eq!(extract("ls -la"), vec![vec!["ls", "-la"]]);
    }

    #[test]
    fn empty_input_is_empty() {
        assert!(parse("").unwrap().commands.is_empty());
        assert!(parse("   ").unwrap().commands.is_empty());
    }

    #[test]
    fn semicolon_splits() {
        let extracted = extract("ls; pwd; whoami");
        assert_eq!(extracted, vec![vec!["ls"], vec!["pwd"], vec!["whoami"]]);
    }

    #[test]
    fn and_or_split() {
        assert_eq!(
            extract("foo && bar || baz"),
            vec![vec!["foo"], vec!["bar"], vec!["baz"]]
        );
    }

    #[test]
    fn pipes_split() {
        assert_eq!(extract("ls | grep foo | wc"), vec![
            vec!["ls"],
            vec!["grep", "foo"],
            vec!["wc"],
        ]);
    }

    #[test]
    fn background_amp_splits() {
        assert_eq!(extract("sleep 1 & echo done"), vec![
            vec!["sleep", "1"],
            vec!["echo", "done"],
        ]);
    }

    #[test]
    fn quoted_strings_preserve_connectors() {
        // The `;` is inside quotes and must NOT split.
        assert_eq!(
            extract(r#"echo "hello; world""#),
            vec![vec!["echo", "hello; world"]]
        );
    }

    #[test]
    fn escaped_quote_is_not_a_close() {
        let parsed = parse(r#"echo "he said \"hi\"""#).unwrap();
        assert_eq!(parsed.commands.len(), 1);
        assert_eq!(parsed.commands[0].argv[0], "echo");
    }

    #[test]
    fn unterminated_quote_fails() {
        assert!(matches!(
            parse(r#"echo "open"#),
            Err(ParseError::UnbalancedQuote('"'))
        ));
        assert!(matches!(
            parse("echo 'open"),
            Err(ParseError::UnbalancedQuote('\''))
        ));
    }

    #[test]
    fn command_substitution_recurses() {
        // The outer command is `echo`, but the inner $(rm -rf /) must
        // surface as its own command for matching.
        let parsed = parse("echo $(rm -rf /)").unwrap();
        let argvs: Vec<&str> = parsed.commands.iter().map(|c| c.binary()).collect();
        assert!(argvs.contains(&"echo"));
        assert!(argvs.contains(&"rm"));
    }

    #[test]
    fn backtick_substitution_recurses() {
        let parsed = parse("foo `rm -rf bar`").unwrap();
        let bins: Vec<&str> = parsed.commands.iter().map(|c| c.binary()).collect();
        assert!(bins.contains(&"foo"));
        assert!(bins.contains(&"rm"));
    }

    #[test]
    fn subshell_recurses() {
        let parsed = parse("(cd / && rm -rf *)").unwrap();
        let bins: Vec<&str> = parsed.commands.iter().map(|c| c.binary()).collect();
        assert!(bins.contains(&"cd"));
        assert!(bins.contains(&"rm"));
    }

    #[test]
    fn env_assignments_stripped() {
        let parsed = parse("FOO=bar BAZ=qux rm -rf /tmp/x").unwrap();
        assert_eq!(parsed.commands[0].binary(), "rm");
    }

    #[test]
    fn time_prefix_stripped() {
        let parsed = parse("time rm -rf foo").unwrap();
        assert_eq!(parsed.commands[0].binary(), "rm");
    }

    #[test]
    fn sudo_prefix_stripped() {
        let parsed = parse("sudo terraform destroy").unwrap();
        assert_eq!(parsed.commands[0].binary(), "terraform");
    }

    #[test]
    fn absolute_path_binary_matched_by_basename() {
        let parsed = parse("/usr/bin/rm -rf /tmp").unwrap();
        let matcher = Parser::with_default_corpus();
        let res = matcher.match_corpus(&parsed);
        assert!(matches!(res, MatchResult::Destructive(_)));
    }

    #[test]
    fn windows_exe_suffix_stripped() {
        // bin_basename strips .exe so corpus matching works for
        // cross-platform commands.
        assert_eq!(bin_basename("C:\\bin\\rm.exe"), "rm");
        assert_eq!(bin_basename("/usr/bin/git"), "git");
        assert_eq!(bin_basename("git"), "git");
    }

    #[test]
    fn rm_matches_path_dependent_pattern() {
        let p = Parser::with_default_corpus();
        let parsed = p.parse("rm -rf /tmp/x").unwrap();
        let m = p.match_corpus(&parsed);
        match m {
            MatchResult::Destructive(hit) => assert_eq!(hit.pattern_id, "rm-recursive"),
            other => panic!("expected destructive, got {other:?}"),
        }
    }

    #[test]
    fn benign_git_status_no_match() {
        let p = Parser::with_default_corpus();
        let parsed = p.parse("git status").unwrap();
        assert_eq!(p.match_corpus(&parsed), MatchResult::Benign);
    }

    #[test]
    fn nested_substitution_under_limit_succeeds() {
        let mut input = String::from("rm -rf foo");
        for _ in 0..5 {
            input = format!("echo $({input})");
        }
        let parsed = parse(&input).unwrap();
        assert!(parsed.commands.iter().any(|c| c.binary() == "rm"));
    }

    #[test]
    fn deep_nesting_hits_recursion_limit() {
        let mut input = String::from("rm");
        for _ in 0..(MAX_RECURSION + 1) {
            input = format!("$({input})");
        }
        assert!(matches!(parse(&input), Err(ParseError::RecursionLimit)));
    }

    #[test]
    fn unbalanced_subshell_fails() {
        assert!(matches!(
            parse("rm -rf $(echo foo"),
            Err(ParseError::UnbalancedCmdSubst)
        ));
        assert!(matches!(parse("(cd /;"), Err(ParseError::UnbalancedSubshell)));
    }

    #[test]
    fn rm_in_quoted_string_does_not_match() {
        // echo "rm -rf /" should NOT match — there's no rm being invoked.
        let p = Parser::with_default_corpus();
        let parsed = p.parse(r#"echo "rm -rf /""#).unwrap();
        assert_eq!(parsed.commands.len(), 1);
        assert_eq!(parsed.commands[0].binary(), "echo");
        assert_eq!(p.match_corpus(&parsed), MatchResult::Benign);
    }

    #[test]
    fn destructive_after_benign_still_matches() {
        let p = Parser::with_default_corpus();
        let parsed = p.parse("echo hi; rm -rf /tmp").unwrap();
        let m = p.match_corpus(&parsed);
        assert!(matches!(m, MatchResult::Destructive(_)));
    }

    #[test]
    fn brace_group_recurses() {
        let p = Parser::with_default_corpus();
        let parsed = p.parse("{ cd /tmp; rm -rf *; }").unwrap();
        let bins: Vec<&str> = parsed.commands.iter().map(|c| c.binary()).collect();
        assert!(bins.contains(&"rm"));
    }
}
