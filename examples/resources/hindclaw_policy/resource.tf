data "hindclaw_policy_document" "agent_access" {
  statement {
    effect  = "allow"
    actions = ["recall", "retain"]
    banks   = ["personal"]

    recall_budget     = "medium"
    recall_max_tokens = 4096
    retain_roles      = ["user", "assistant"]
    retain_tags       = ["conversation"]
  }
}

resource "hindclaw_policy" "agent_access" {
  id           = "agent-access"
  display_name = "Agent Access Policy"
  document     = data.hindclaw_policy_document.agent_access.json
}
