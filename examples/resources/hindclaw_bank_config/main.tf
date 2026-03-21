resource "hindclaw_bank_config" "agent_alpha" {
  bank_id = "agent-alpha"
  config = jsonencode({
    llm_model    = "gpt-4o"
    llm_provider = "openai"
    chunk_size   = 512
  })
}
