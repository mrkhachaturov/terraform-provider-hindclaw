resource "hindclaw_service_account" "ci_bot" {
  id                = "ci-bot"
  owner_user_id     = "alice"
  display_name      = "CI Bot"
  scoping_policy_id = hindclaw_policy.ci_readonly.id
}
