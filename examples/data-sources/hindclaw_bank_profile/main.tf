data "hindclaw_bank_profile" "alpha" {
  bank_id = "agent-alpha"
}

output "bank_name" {
  value = data.hindclaw_bank_profile.alpha.name
}

output "bank_mission" {
  value = data.hindclaw_bank_profile.alpha.mission
}
