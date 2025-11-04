{{- /* Copyright 2025 Andrew Vasilyev */ -}}
{{- /* SPDX-License-Identifier: Apache-2.0 */ -}}

{{- /* Vault Agent template for PKI Intermediate CA certificate */ -}}
{{- /* This certificate is used by services to validate peer mTLS certificates */ -}}

{{- with secret "pki_int/cert/ca" -}}
{{ .Data.certificate }}
{{- end -}}
