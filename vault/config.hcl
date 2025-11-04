# Vault production configuration
# WARNING: TLS is disabled for internal-only access
# Enable TLS if exposing Vault outside Docker network
listener "tcp" {
  address         = "0.0.0.0:8200"
  cluster_address = "0.0.0.0:8201"
  tls_disable     = 1
}

storage "file" {
  path = "/vault/data"
}

ui            = true
# Vault UI accessed via token authentication (dev: token=root)
disable_mlock = true
api_addr      = "http://vault:8200"
cluster_addr  = "http://vault:8201"
