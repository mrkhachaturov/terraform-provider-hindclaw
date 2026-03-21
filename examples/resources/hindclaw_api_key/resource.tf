resource "hindclaw_api_key" "alice_key" {
  user_id     = "alice"
  description = "Primary API key"
}

output "api_key" {
  value     = hindclaw_api_key.alice_key.api_key
  sensitive = true
}
