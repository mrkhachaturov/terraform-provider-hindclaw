# terraform-provider-hindclaw

Terraform provider for managing [Hindsight](https://hindsight.vectorize.io) memory banks and [Hindclaw](https://hindclaw.pro) access control.

## Requirements

- Go 1.21+
- Terraform 1.0+

## Resources

| Resource | Description |
|----------|-------------|
| `hindclaw_user` | Manage users (id, display_name, email) |
| `hindclaw_group` | Manage groups with permission defaults (recall, retain, tags, roles, strategies) |

## Provider Configuration

```hcl
provider "hindclaw" {
  api_url = "https://your-hindsight-server"  # or HINDCLAW_API_URL env var
  api_key = "your-api-key"                   # or HINDCLAW_API_KEY env var
}
```

## Example Usage

```hcl
resource "hindclaw_user" "ruben" {
  id           = "ruben"
  display_name = "Ruben Khachaturov"
  email        = "ruben@example.com"
}

resource "hindclaw_group" "agents" {
  id           = "agents"
  display_name = "AI Agents"
  recall       = true
  retain       = true
  retain_tags  = ["agent", "internal"]
  recall_budget = "mid"
}
```

## Development

```bash
# Build
make build

# Install to local Terraform plugins
make install

# Run tests
make test

# Run acceptance tests (requires running Hindclaw server)
TF_ACC=1 make testacc
```

### Dev Override

Add to `~/.terraformrc` to use the local build:

```hcl
provider_installation {
  dev_overrides {
    "hindclaw.pro/mrkhachaturov/hindclaw" = "/path/to/hindclaw-terraform"
  }
  direct {}
}
```

## License

See [LICENSE](../LICENSE).
