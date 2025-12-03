{{- if and (not .Values.existingSecret) .Values.secret.create }}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "logistic.fullname" . }}
  labels:
    {{- include "logistic.labels" . | nindent 4 }}
type: Opaque
data:
  freightcom-api-key: {{ .Values.secret.freightcomApiKey | b64enc | quote }}
  canadapost-api-key: {{ .Values.secret.canadapostApiKey | b64enc | quote }}
  canadapost-account-id: {{ .Values.secret.canadapostAccountId | b64enc | quote }}
  purolator-username: {{ .Values.secret.purolatorUsername | b64enc | quote }}
  purolator-password: {{ .Values.secret.purolatorPassword | b64enc | quote }}
{{- end }}
