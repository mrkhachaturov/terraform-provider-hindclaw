# terraform-provider-hindclaw

[![CI](https://github.com/mrkhachaturov/terraform-provider-hindclaw/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/mrkhachaturov/terraform-provider-hindclaw/actions/workflows/ci.yml)
[![Release](https://github.com/mrkhachaturov/terraform-provider-hindclaw/actions/workflows/release.yml/badge.svg)](https://github.com/mrkhachaturov/terraform-provider-hindclaw/actions/workflows/release.yml)
[![License: MPL-2.0](https://img.shields.io/badge/license-MPL--2.0-brightgreen.svg)](LICENSE)

`terraform-provider-hindclaw` manages Hindclaw access control and Hindsight memory-bank resources from Terraform and OpenTofu. It is built on the Terraform Plugin Framework and is set up for signed GitHub releases, generated documentation, runnable examples, and public registry publication.

## Highlights

- One provider for both Hindclaw authorization and Hindsight memory-bank management.
- Full CRUD coverage for users, groups, bank permissions, banks, directives, mental models, and webhooks.
- Sensitive and write-only handling for API keys, plus 404-aware refresh/delete behavior across resources.
- Generated docs, acceptance-test scaffolding, and release automation already included in the repository.

## Compatibility

| Component | Version |
| --- | --- |
| Go | `1.26+` |
| Terraform | `>= 1.0` |
| OpenTofu | `>= 1.6` |
| Provider protocol | `6.0` |

## Provider Source

Use the public provider source address:

```hcl
terraform {
  required_providers {
    hindclaw = {
      source = "mrkhachaturov/hindclaw"
    }
  }
}
```

The binary serves `registry.terraform.io/mrkhachaturov/hindclaw`, which matches Terraform Registry publication and also works cleanly with OpenTofu provider addressing.

## Resource Coverage

| Area | Resources |
| --- | --- |
| Identity and access | `hindclaw_user`, `hindclaw_user_channel`, `hindclaw_group`, `hindclaw_group_membership`, `hindclaw_bank_permission`, `hindclaw_strategy_scope`, `hindclaw_api_key` |
| Memory banks | `hindclaw_bank`, `hindclaw_bank_config` |
| Hindsight intelligence | `hindclaw_mental_model`, `hindclaw_directive`, `hindclaw_webhook` |

| Data source | Purpose |
| --- | --- |
| `hindclaw_resolved_permissions` | Resolve effective permissions for a user/context |
| `hindclaw_bank_profile` | Read a bank profile |
| `hindclaw_banks` | List available banks |

Generated reference docs are in [docs/](docs/), and runnable examples are in [examples/](examples/).

## Quick Start

```hcl
terraform {
  required_providers {
    hindclaw = {
      source = "mrkhachaturov/hindclaw"
    }
  }
}

variable "hindclaw_api_key" {
  type      = string
  sensitive = true
}

provider "hindclaw" {
  api_url = "https://hindsight.example.internal"
  api_key = var.hindclaw_api_key
}

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

resource "hindclaw_bank_permission" "agents_alpha" {
  bank_id    = hindclaw_bank.alpha.bank_id
  scope_type = "group"
  scope_id   = hindclaw_group.agents.id
  recall     = true
  retain     = true
}
```

## Authentication

The provider accepts configuration directly or through environment variables:

```hcl
provider "hindclaw" {
  api_url = "https://your-hindsight-server"
  api_key = var.hindclaw_api_key
}
```

- `HINDCLAW_API_URL`
- `HINDCLAW_API_KEY`

## Documentation and Examples

- Provider docs: [docs/index.md](docs/index.md)
- Resource docs: [docs/resources/](docs/resources/)
- Data source docs: [docs/data-sources/](docs/data-sources/)
- Examples index: [examples/README.md](examples/README.md)
- Starter provider config: [examples/provider/provider.tf](examples/provider/provider.tf)

## Local Development

Build the provider:

```bash
make build
```

Install it into the local plugin directory:

```bash
make install VERSION=0.0.0-dev
```

For local development without a published release, add a dev override to `~/.terraformrc` for Terraform or `~/.tofurc` for OpenTofu:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/mrkhachaturov/hindclaw" = "/absolute/path/to/hindclaw-terraform"
  }
  direct {}
}
```

Then:

```bash
make build
cd examples/provider
terraform init
terraform plan
```

## Common Commands

```bash
make fmt
make vet
make test
make build
make generate
make release-check
```

Acceptance tests require a live Hindsight server with Hindclaw enabled:

```bash
export HINDCLAW_API_URL="https://hindsight.example.internal"
export HINDCLAW_API_KEY="hc_test_xxxxx"
make testacc
```

## Releases

This repository includes automated CI, generated documentation, and tagged GitHub releases for provider distribution. Maintainer-oriented release steps and registry publication notes are documented in [RELEASING.md](RELEASING.md).

## Repository Layout

- [internal/provider](internal/provider) contains the provider, resources, data sources, and tests.
- [examples/](examples/) contains runnable examples for the provider, all resources, and all data sources.
- [docs/](docs/) contains generated documentation.
- [GNUmakefile](GNUmakefile) provides local build, test, install, generate, and release-check commands.

## Security

See [SECURITY.md](SECURITY.md) for vulnerability reporting guidance.

## Licensing and Attribution

This repository is licensed under [MPL-2.0](LICENSE).

Upstream Hindsight and the surrounding Hindclaw workspace are MIT-licensed. For attribution and upstream license references, see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
