{{- define "octelium-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "octelium-operator.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := include "octelium-operator.name" . -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "octelium-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "octelium-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "octelium-operator.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | quote }}
{{ include "octelium-operator.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "octelium-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "octelium-operator.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- required "serviceAccount.name is required when serviceAccount.create=false" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "octelium-operator.clusterRoleName" -}}
{{- $name := include "octelium-operator.fullname" . -}}
{{- printf "%s-%s" ($name | trunc 52 | trimSuffix "-") (printf "%s/%s" .Release.Namespace $name | sha256sum | trunc 10) -}}
{{- end -}}

{{- define "octelium-operator.image" -}}
{{- if .Values.image.digest -}}
{{- printf "%s@%s" .Values.image.repository .Values.image.digest -}}
{{- else -}}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) -}}
{{- end -}}
{{- end -}}

{{- define "octelium-operator.leaderElectionName" -}}
{{- printf "%s-leader-election" (include "octelium-operator.fullname" . | trunc 47 | trimSuffix "-") -}}
{{- end -}}

{{- define "octelium-operator.metricsName" -}}
{{- printf "%s-metrics" (include "octelium-operator.fullname" . | trunc 55 | trimSuffix "-") -}}
{{- end -}}
