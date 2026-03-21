resource "hindclaw_webhook" "notify" {
  bank_id     = "agent-alpha"
  url         = "https://hooks.example.com/hindsight"
  event_types = ["memory.retained", "memory.recalled"]
  enabled     = true
}
