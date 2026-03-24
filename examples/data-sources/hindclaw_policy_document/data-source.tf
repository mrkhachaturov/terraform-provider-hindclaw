data "hindclaw_policy_document" "example" {
  statement {
    effect  = "allow"
    actions = ["recall", "retain"]
    banks   = ["shared"]

    recall_budget     = "high"
    recall_max_tokens = 8192
    retain_roles      = ["user", "assistant"]
    retain_tags       = ["meeting-notes"]
    retain_strategy   = "detailed"
  }

  statement {
    effect  = "deny"
    actions = ["retain"]
    banks   = ["restricted"]
  }
}

output "policy_json" {
  value = data.hindclaw_policy_document.example.json
}
