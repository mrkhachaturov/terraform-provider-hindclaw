resource "hindclaw_strategy_scope" "alpha_topic" {
  bank_id     = "agent-alpha"
  scope_type  = "topic"
  scope_value = "12345"
  strategy    = "detailed"
}
