resource "hindclaw_mental_model" "user_profile" {
  bank_id      = "agent-alpha"
  name         = "User Profile"
  source_query = "What do we know about the user?"
  tags         = ["profile", "core"]
  max_tokens   = 4096

  trigger = {
    refresh_after_consolidation = true
    fact_types                  = ["experience", "observation"]
  }
}
