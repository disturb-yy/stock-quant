{{/* 展开 chart 名称。 */}}
{{- define "stock-quant.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* 创建带发布名称的资源名称。 */}}
{{- define "stock-quant.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/* 创建 chart 标签值。 */}}
{{- define "stock-quant.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* 通用标签。 */}}
{{- define "stock-quant.labels" -}}
helm.sh/chart: {{ include "stock-quant.chart" . }}
{{ include "stock-quant.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/* 选择器标签。 */}}
{{- define "stock-quant.selectorLabels" -}}
app.kubernetes.io/name: {{ include "stock-quant.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/* ServiceAccount 名称。 */}}
{{- define "stock-quant.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "stock-quant.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
