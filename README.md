# terraform-provider-hindclaw

Terraform provider for managing [Hindsight](https://hindsight.vectorize.io) memory banks and [Hindclaw](https://hindclaw.pro) access control. Compatible with Terraform and OpenTofu.

## Requirements

- Go 1.21+
- Terraform 1.0+ or OpenTofu 1.6+

## Resources

| Resource | Description |
|----------|-------------|
| `hindclaw_user` | Manage users (id, display_name, email) |
| `hindclaw_user_channel` | Map channel senders to users |
| `hindclaw_group` | Manage groups with permission defaults |
| `hindclaw_group_membership` | Manage user-to-group memberships |
| `hindclaw_bank_permission` | Per-bank permission overrides for users/groups |
| `hindclaw_strategy_scope` | Retain strategy bindings per bank scope |
| `hindclaw_api_key` | API keys (secret available at creation only) |
| `hindclaw_bank` | Hindsight memory bank identity and profile |
| `hindclaw_bank_config` | Bank configuration overrides (JSON) |
| `hindclaw_mental_model` | Mental models for a bank |
| `hindclaw_directive` | Directives for a bank |
| `hindclaw_webhook` | Webhooks for bank events |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `hindclaw_resolved_permissions` | Resolve effective permissions for a context |
| `hindclaw_bank_profile` | Read a bank's profile |
| `hindclaw_banks` | List all banks |

## Provider Configuration

```hcl
provider "hindclaw" {
  api_url = "https://your-hindsight-server"  # or HINDCLAW_API_URL env var
  api_key = "your-api-key"                   # or HINDCLAW_API_KEY env var
}
```

## Example Usage

```hcl
resource "hindclaw_user" "alice" {
  id           = "alice"
  display_name = "Alice Smith"
  email        = "alice@example.com"
}

resource "hindclaw_group" "agents" {
  id           = "agents"
  display_name = "AI Agents"
  recall       = true
  retain       = true
  retain_tags  = ["agent", "internal"]
}

resource "hindclaw_bank" "alpha" {
  bank_id = "agent-alpha"
  name    = "Agent Alpha"
  mission = "Strategic mentor and advisor"
}
```

## Development

### Build and Install

```bash
make build
make install
```

### Local Testing (Dev Override)

Add to `~/.terraformrc` (Terraform) or `~/.tofurc` (OpenTofu):

```hcl
provider_installation {
  dev_overrides {
    "hindclaw.pro/mrkhachaturov/hindclaw" = "/path/to/hindclaw-terraform"
  }
  direct {}
}
```

Then:

```bash
make build
cd examples/provider
terraform plan   # or: tofu plan
```

### Running Tests

```bash
# Unit tests (no server needed)
make test

# Acceptance tests (requires running Hindsight server with hindclaw extensions)
export HINDCLAW_API_URL="https://hindsight.home.local"
export HINDCLAW_API_KEY="hc_test_xxxxx"
make testacc
```

### Generate Documentation

```bash
go generate ./...
ls docs/
```

## License

See [LICENSE](../LICENSE).
