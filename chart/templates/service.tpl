apiVersion: v1
kind: Service
metadata:
  name: {{ include "logistic.fullname" . }}
  labels:
    {{- include "logistic.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "logistic.selectorLabels" . | nindent 4 }}
