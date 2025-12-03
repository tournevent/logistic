apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "logistic.fullname" . }}
  labels:
    {{- include "logistic.labels" . | nindent 4 }}
spec:
  {{- if not .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "logistic.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      {{- with .Values.podAnnotations }}
      annotations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      labels:
        {{- include "logistic.labels" . | nindent 8 }}
        {{- with .Values.podLabels }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "logistic.serviceAccountName" . }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: {{ .Chart.Name }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: 80
              protocol: TCP
          livenessProbe:
            {{- toYaml .Values.livenessProbe | nindent 12 }}
          readinessProbe:
            {{- toYaml .Values.readinessProbe | nindent 12 }}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          env:
            - name: PORT
              value: {{ .Values.env.port | quote }}
            - name: LOG_LEVEL
              value: {{ .Values.env.logLevel | quote }}
            - name: FREIGHTCOM_ENABLED
              value: {{ .Values.env.freightcomEnabled | quote }}
            - name: FREIGHTCOM_USE_MOCK
              value: {{ .Values.env.freightcomUseMock | quote }}
            - name: CANADAPOST_ENABLED
              value: {{ .Values.env.canadapostEnabled | quote }}
            - name: CANADAPOST_USE_MOCK
              value: {{ .Values.env.canadapostUseMock | quote }}
            - name: PUROLATOR_ENABLED
              value: {{ .Values.env.purolatorEnabled | quote }}
            - name: PUROLATOR_USE_MOCK
              value: {{ .Values.env.purolatorUseMock | quote }}
            - name: OTEL_ENABLED
              value: {{ .Values.env.otelEnabled | quote }}
            {{- if or .Values.existingSecret .Values.secret.create }}
            - name: FREIGHTCOM_API_KEY
              valueFrom:
                secretKeyRef:
                  name: {{ include "logistic.secretName" . }}
                  key: freightcom-api-key
            - name: CANADAPOST_API_KEY
              valueFrom:
                secretKeyRef:
                  name: {{ include "logistic.secretName" . }}
                  key: canadapost-api-key
            - name: CANADAPOST_ACCOUNT_ID
              valueFrom:
                secretKeyRef:
                  name: {{ include "logistic.secretName" . }}
                  key: canadapost-account-id
            - name: PUROLATOR_USERNAME
              valueFrom:
                secretKeyRef:
                  name: {{ include "logistic.secretName" . }}
                  key: purolator-username
            - name: PUROLATOR_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ include "logistic.secretName" . }}
                  key: purolator-password
            {{- end }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
