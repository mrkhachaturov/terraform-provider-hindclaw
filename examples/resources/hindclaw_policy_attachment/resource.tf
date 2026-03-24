resource "hindclaw_policy_attachment" "agents_default" {
  policy_id      = "default-policy"
  principal_type = "group"
  principal_id   = "agents"
  priority       = 10
}

resource "hindclaw_policy_attachment" "yoda_override" {
  policy_id      = "admin-policy"
  principal_type = "user"
  principal_id   = "yoda"
  priority       = 100
}
