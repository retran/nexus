{{- /* Copyright 2025 Andrew Vasilyev */ -}}
{{- /* SPDX-License-Identifier: Apache-2.0 */ -}}

{{- /* Vault Agent template for service mTLS private key */ -}}
{{- /* Renders the private key corresponding to the service certificate */ -}}
{{- /* Key is automatically rotated together with the certificate */ -}}
{{- /* Requires PKI_ROLE and COMMON_NAME environment variables */ -}}

{{- $role := env "PKI_ROLE" -}}
{{- $cn := env "COMMON_NAME" -}}
{{- $path := printf "pki_int/issue/%s" $role -}}
{{- with pkiCert $path (printf "common_name=%s" $cn) "ttl=1h" -}}
{{ .Key }}
{{- end -}}
