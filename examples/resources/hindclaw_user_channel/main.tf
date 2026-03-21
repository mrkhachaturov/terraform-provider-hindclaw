resource "hindclaw_user_channel" "alice_telegram" {
  user_id          = "alice"
  channel_provider = "telegram"
  sender_id        = "123456789"
}
