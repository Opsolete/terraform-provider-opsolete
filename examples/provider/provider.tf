terraform {
  required_providers {
    opsolete = {
      source = "opsolete/opsolete"
    }
  }
}

# A service token: Settings → API tokens in Opsolete, or `opsolete api-token terraform` on the server.
# Or leave both out and set OPSOLETE_URL and OPSOLETE_TOKEN.
provider "opsolete" {
  url   = "https://opsolete.example.com"
  token = var.opsolete_token
}

variable "opsolete_token" {
  type      = string
  sensitive = true
}
