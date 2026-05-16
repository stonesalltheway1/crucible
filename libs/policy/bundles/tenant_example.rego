# Example tenant override.
#
# Tenants ship a TenantBundle (JSON) that contains one or more `.rego` files
# whose `package` MUST be `crucible.promotion.tenant`. The promotion gate
# evaluates the tenant entrypoint AFTER the default; the merged decision is
# the conjunction (allow AND, needs_human OR) of both.
#
# This file is for documentation + tests. Customers wrap analogous rules in
# their own bundles and sign them with `crucible policy sign`.

package crucible.promotion.tenant

import rego.v1

# Default-deny mirror — tenants opt into "tenant has nothing to say" by
# returning {"allow": true} explicitly. Anything else is treated as a deny.
default decision := {
	"allow": true,
	"needs_human": false,
	"reasons": [],
	"require_codeowner": false,
	"approver_groups": [],
	"require_n_approvers": 0,
	"auto_approve": false,
}

# Tenant rule: deploys to prod-eu must be approved by an EU-resident OIDC
# subject. The gate fills `input.context.geo` from the agent worker pool.
decision := {
	"allow": false,
	"needs_human": true,
	"reasons": ["tenant: prod-eu deploys require an EU-based approver"],
	"require_codeowner": false,
	"approver_groups": ["@eu-on-call"],
	"require_n_approvers": 1,
	"auto_approve": false,
} if {
	input.context.geo == "eu"
	not eu_approver_present
}

eu_approver_present if {
	some a in input.approvals
	endswith(a.approver_oidc_subject, "@acme-eu.com")
}

# Tenant rule: any change touching billing/ requires @payments-leads.
decision := {
	"allow": false,
	"needs_human": true,
	"reasons": ["tenant: billing/ touch requires @payments-leads"],
	"require_codeowner": true,
	"approver_groups": ["@payments-leads"],
	"require_n_approvers": 1,
	"auto_approve": false,
} if {
	some f in input.files_changed
	startswith(f.path, "src/billing/")
	not has_payments_lead_approval
}

has_payments_lead_approval if {
	some a in input.approvals
	a.group == "@payments-leads"
}
