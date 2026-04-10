FROM --platform=$BUILDPLATFORM golang:1.25.9 AS build
ARG TARGETARCH
WORKDIR /go/src/app
COPY . .
RUN GOARCH=$TARGETARCH make

FROM scratch
COPY *.html ./
COPY *.png ./
COPY *.js ./
COPY *.ico ./
COPY *.css ./
COPY --from=build /go/src/app/rollouts-demo /rollouts-demo

ARG COLOR
ENV COLOR=${COLOR}
ARG ERROR_RATE
ENV ERROR_RATE=${ERROR_RATE}
ARG LATENCY
ENV LATENCY=${LATENCY}

# OTLP metrics push endpoint — override at runtime via env or k8s manifest
ENV OTEL_EXPORTER_OTLP_ENDPOINT=""

ENTRYPOINT [ "/rollouts-demo" ]
