# Copyright 2025 Andrew Vasilyev
# SPDX-License-Identifier: Apache-2.0

# Vault Agent configuration for automatic mTLS certificate provisioning
# This config uses AppRole authentication and Consul Template to render
# certificates, keys, and secrets to files for service consumption

vault {
  address = "http://vault:8200"
}

# Auto-authentication using AppRole method
# Role ID and Secret ID are provided via files mounted from host
auto_auth {
  method {
    type = "approle"

    config = {
      role_id_file_path                   = "/vault/role-id"
      secret_id_file_path                 = "/vault/secret-id"
      remove_secret_id_file_after_reading = false
    }
  }
}

# Template configuration for all templates
template_config {
  # Re-render static secrets every 5 minutes
  static_secret_render_interval = "5m"

  # Exit on retry failures to allow container orchestrator to restart
  exit_on_retry_failure = true

  # Renew leases when 90% of TTL is reached (certificates)
  lease_renewal_threshold = 0.9
}

# Template: CA Certificate
# Renders the Vault PKI Intermediate CA certificate
# This is used by services to validate peer certificates
template {
  source      = "/config/ca.tpl"
  destination = "/secrets/vault-ca.pem"
  perms       = "0644"

  # Signal readiness after CA is rendered
  command = "touch /secrets/.ca-ready"
}

# Template: Service TLS Certificate
# Issues a new certificate from Vault PKI engine using the service's role
# Certificate is automatically renewed when 90% of TTL is reached
template {
  source      = "/config/cert.tpl"
  destination = "/secrets/tls.crt"
  perms       = "0644"

  # Signal readiness after certificate is rendered
  command = "touch /secrets/.cert-ready"
}

# Template: Service TLS Private Key
# Renders the private key corresponding to the service certificate
# Key is automatically rotated together with the certificate
template {
  source      = "/config/key.tpl"
  destination = "/secrets/tls.key"
  perms       = "0600"

  # Signal readiness after key is rendered
  command = "touch /secrets/.key-ready"
}

# Template: Webhook Secret (optional, for kratos-agent only)
# Renders the shared webhook secret from KV store
# This template will fail if the service doesn't have access to webhook path
# For services other than kratos, this template should be omitted
# template {
#   source      = "/config/webhook.tpl"
#   destination = "/secrets/webhook-secret"
#   perms       = "0600"
#
#   command = "touch /secrets/.webhook-ready"
# }
