{{- /* Copyright 2025 Andrew Vasilyev */ -}}
{{- /* SPDX-License-Identifier: Apache-2.0 */ -}}

{{- /* Vault Agent template for Kratos webhook secret */ -}}
{{- /* Renders the current webhook secret from KV store */ -}}
{{- /* Used by Kratos to sign webhook requests to system API */ -}}

{{- with secret "kv/data/shared/webhook" -}}
{{ .Data.data.current }}
{{- end -}}
