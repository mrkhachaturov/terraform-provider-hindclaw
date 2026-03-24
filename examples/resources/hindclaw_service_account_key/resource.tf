resource "hindclaw_service_account_key" "ci_bot_key" {
  service_account_id = hindclaw_service_account.ci_bot.id
  description        = "Primary API key"
}

output "ci_bot_api_key" {
  value     = hindclaw_service_account_key.ci_bot_key.api_key
  sensitive = true
}
