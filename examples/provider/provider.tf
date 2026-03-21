terraform {
  required_providers {
    hindclaw = {
      source = "mrkhachaturov/hindclaw"
    }
  }
}

provider "hindclaw" {
  api_url = "https://hindsight.home.local"
  api_key = var.hindclaw_api_key
}

variable "hindclaw_api_key" {
  type      = string
  sensitive = true
}
