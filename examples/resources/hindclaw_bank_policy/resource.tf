resource "hindclaw_bank_policy" "alpha" {
  bank_id  = "agent-alpha"
  document = jsonencode({
    version = "1"
    statements = [
      {
        effect    = "allow"
        actions   = ["recall", "retain"]
        resources = ["*"]
      }
    ]
  })
}
