{{/*
Common labels applied to all resources.
*/}}
{{- define "qeet-notify.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Image reference helper.
*/}}
{{- define "qeet-notify.image" -}}
{{ .Values.image.registry }}/{{ .name }}:{{ .Values.image.tag }}
{{- end }}
