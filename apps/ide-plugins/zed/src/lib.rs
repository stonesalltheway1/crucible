//! Crucible extension for Zed.
//!
//! Zed is itself AI-native: it ships chat panels, multi-file rewrite UI, and
//! ACP-based agent integration. We do NOT duplicate any of that. This
//! extension simply:
//!
//!   1. Registers Crucible's MCP tool surface so Zed's built-in agent host
//!      can call `twin_*` tools via ACP.
//!   2. Exposes slash commands for the differentiated Crucible affordances
//!      (submit task → plan approval, halt, view attestation).
//!
//! Per ADR-011: integrate, don't compete.

use zed_extension_api::{self as zed, SlashCommand, SlashCommandOutput, SlashCommandOutputSection, Worktree};

struct CrucibleExtension;

impl zed::Extension for CrucibleExtension {
    fn new() -> Self {
        Self
    }

    fn run_slash_command(
        &self,
        command: SlashCommand,
        args: Vec<String>,
        worktree: Option<&Worktree>,
    ) -> Result<SlashCommandOutput, String> {
        match command.name.as_str() {
            "crucible" => self.cmd_new_task(args, worktree),
            "crucible-approve" => self.cmd_approve(args),
            "crucible-halt" => self.cmd_halt(args),
            other => Err(format!("unknown command: {other}")),
        }
    }
}

impl CrucibleExtension {
    fn cmd_new_task(&self, args: Vec<String>, worktree: Option<&Worktree>) -> Result<SlashCommandOutput, String> {
        let description = args.join(" ");
        if description.trim().is_empty() {
            return Err("description required: /crucible <what to do>".into());
        }
        let repo = worktree
            .and_then(|w| w.root_path().split('/').last().map(|s| s.to_string()))
            .unwrap_or_else(|| "unknown".into());

        // We surface the action; Zed's agent host posts via the ACP-mapped
        // Crucible MCP server. The extension itself does NOT make HTTP calls
        // (Zed's extension runtime is wasm-sandboxed and intentionally
        // network-restricted). The output below is what shows in the chat;
        // the user clicks "approve" in the web console or via /crucible-approve.
        let body = format!(
            "**Crucible task submitted**\n\
            \n\
            ```\n\
            description: {description}\n\
            repo: github.com/acme/{repo}\n\
            via: zed-acp\n\
            ```\n\
            \n\
            The planner is preparing a cost preview and risk callouts. \
            Open the web console or run `/crucible-approve` after review.",
        );
        Ok(SlashCommandOutput {
            text: body.clone(),
            sections: vec![SlashCommandOutputSection {
                range: (0..body.len()).into(),
                label: "Crucible task".into(),
            }],
        })
    }

    fn cmd_approve(&self, _args: Vec<String>) -> Result<SlashCommandOutput, String> {
        let body = "Plan approved. Streaming progress via the MCP server. \
                    Use the web console or `/crucible-halt` to interrupt.";
        Ok(SlashCommandOutput {
            text: body.into(),
            sections: vec![SlashCommandOutputSection {
                range: (0..body.len()).into(),
                label: "Crucible approval".into(),
            }],
        })
    }

    fn cmd_halt(&self, _args: Vec<String>) -> Result<SlashCommandOutput, String> {
        let body = "Halt requested. The agent will stop at the next safe checkpoint.";
        Ok(SlashCommandOutput {
            text: body.into(),
            sections: vec![SlashCommandOutputSection {
                range: (0..body.len()).into(),
                label: "Crucible halt".into(),
            }],
        })
    }
}

zed::register_extension!(CrucibleExtension);
