resource "hindclaw_directive" "greeting" {
  bank_id  = "agent-alpha"
  name     = "Greeting Style"
  content  = "Always greet the user warmly."
  priority = 10
  tags     = ["personality"]
}
