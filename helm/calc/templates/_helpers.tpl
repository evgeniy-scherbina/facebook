{{/*
Common labels shared by every subservice in the calc chart.
Per-service selector label (app: <name>) is added inline in each template,
since selectors must be stable and service-specific.
*/}}
{{- define "calc.commonLabels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/part-of: calc
{{- end -}}
