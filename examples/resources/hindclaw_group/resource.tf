resource "hindclaw_group" "agents" {
  id            = "agents"
  display_name  = "AI Agents"
  recall        = true
  retain        = true
  retain_tags   = ["agent", "internal"]
  recall_budget = "mid"
}
