{{/*
Common labels + helpers shared across the umbrella chart.
*/}}

{{- define "crucible.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
crucible.dev/edition: {{ .Values.crucible.edition | default "enterprise" }}
crucible.dev/topology: {{ .Values.crucible.topology | default "vpc" }}
{{- end -}}

{{- define "crucible.image" -}}
{{ .Values.global.imageRegistry }}/{{ .imageName }}:{{ .Values.global.imageTag }}
{{- end -}}

{{- define "crucible.envFromGlobal" -}}
- name: CRUCIBLE_DOMAIN
  value: {{ .Values.global.domain | quote }}
- name: CRUCIBLE_TOPOLOGY
  value: {{ .Values.crucible.topology | quote }}
{{- end -}}
