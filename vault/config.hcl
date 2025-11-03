# Vault development configuration
listener "tcp" {
  address     = "0.0.0.0:8200"
  cluster_address = "0.0.0.0:8201"
  tls_disable = 1
}

storage "file" {
  path = "/vault/data"
}

ui            = true
# OIDC auth for the Vault UI is configured via scripts/init-oidc.sh
disable_mlock = true
api_addr      = "http://vault:8200"
cluster_addr  = "http://vault:8201"
