data "hindclaw_banks" "all" {}

output "bank_ids" {
  value = [for b in data.hindclaw_banks.all.banks : b.bank_id]
}
