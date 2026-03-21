resource "hindclaw_bank_permission" "agents_alpha" {
  bank_id    = "agent-alpha"
  scope_type = "group"
  scope_id   = "agents"
  recall     = true
  retain     = true
}
