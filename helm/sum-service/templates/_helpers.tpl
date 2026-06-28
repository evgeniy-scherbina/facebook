{{/*
Resource name — intentionally FIXED to the chart name (sum-service) and NOT
prefixed with the release name. Other services resolve this via the DNS name
http://sum-service, so the Service name must stay stable across installs.
Override with fullnameOverride only if you know what you're doing.
*/}}
{{- define "sum-service.fullname" -}}
{{- default .Chart.Name .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels — for humans/tooling (kubectl -l, dashboards).
*/}}
{{- define "sum-service.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end -}}

{{/*
Selector labels — the functional ones that tie Service -> Pods.
Kept as `app: sum-service` to match the rest of the app's convention.
Must be stable (never change on an existing Deployment — selector is immutable).
*/}}
{{- define "sum-service.selectorLabels" -}}
app: {{ include "sum-service.fullname" . }}
{{- end -}}
