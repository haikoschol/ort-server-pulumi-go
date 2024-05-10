ui = true
cluster_name = "vault-integrated-storage"
listener "tcp" {
  address = "[::]:8200"
  cluster_address = "[::]:8201"
  disable_mlock = true
  tls_disable = true
  tls_disable_client_certs = true
#   tls_cert_file = "/vault/userconfig/tls-server/vault.pem"
#   tls_key_file = "/vault/userconfig/tls-server/vault-key.pem"
#   tls_client_ca_file = "/vault/userconfig/tls-server/vault.pem"
}

storage "raft" {
  path = "/vault/data"
  retry_join {
    leader_api_addr = "http://vault-0.vault-internal:8200"
#     leader_ca_cert_file = "/vault/userconfig/tls-ca/ca.pem"
#     leader_client_cert_file = "/vault/userconfig/tls-server/vault.pem"
#     leader_client_key_file = "/vault/userconfig/tls-server/vault-key.pem"
  }
  retry_join {
    leader_api_addr = "http://vault-1.vault-internal:8200"
#     leader_ca_cert_file = "/vault/userconfig/tls-ca/ca.pem"
#     leader_client_cert_file = "/vault/userconfig/tls-server/vault.pem"
#     leader_client_key_file = "/vault/userconfig/tls-server/vault-key.pem"
  }
  retry_join {
    leader_api_addr = "http://vault-2.vault-internal:8200"
#     leader_ca_cert_file = "/vault/userconfig/tls-ca/ca.pem"
#     leader_client_cert_file = "/vault/userconfig/tls-server/vault.pem"
#     leader_client_key_file = "/vault/userconfig/tls-server/vault-key.pem"
  }
#   retry_join {
#     leader_api_addr = "http://vault-3.vault-internal:8200"
#     leader_ca_cert_file = "/vault/userconfig/tls-ca/ca.pem"
#     leader_client_cert_file = "/vault/userconfig/tls-server/vault.pem"
#     leader_client_key_file = "/vault/userconfig/tls-server/vault-key.pem"
#   }
#   retry_join {
#     leader_api_addr = "http://vault-4.vault-internal:8200"
#     leader_ca_cert_file = "/vault/userconfig/tls-ca/ca.pem"
#     leader_client_cert_file = "/vault/userconfig/tls-server/vault.pem"
#     leader_client_key_file = "/vault/userconfig/tls-server/vault-key.pem"
#   }
}
