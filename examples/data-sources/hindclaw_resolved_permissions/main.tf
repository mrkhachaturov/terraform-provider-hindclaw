data "hindclaw_resolved_permissions" "check" {
  bank   = "agent-alpha"
  sender = "telegram:123456789"
  agent  = "agent-alpha"
}

output "can_recall" {
  value = data.hindclaw_resolved_permissions.check.recall
}

output "can_retain" {
  value = data.hindclaw_resolved_permissions.check.retain
}
