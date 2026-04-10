# Argo Rollouts Demo Application

A fork of [argoproj/rollouts-demo](https://github.com/argoproj/rollouts-demo) with OpenTelemetry metrics built in. The app pushes HTTP request duration and in-flight request counts to any OTLP-compatible backend (e.g. VictoriaMetrics, Grafana Alloy, Datadog) via the `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable. RPS, latency percentiles, and error rates can be derived from the histogram at query time.

![img](./demo.png)

## Examples

The following examples are provided:

| Example | Description |
|---------|-------------|
| [Canary](examples/canary) | Rollout which uses the canary update strategy |
| [Blue-Green](examples/blue-green) |  Rollout which uses the blue-green update strategy |
| [Canary Analysis](examples/analysis) | Rollout which performs canary analysis as part of the update. Uses the prometheus metric provider. |
| [Experiment](examples/experiment) | Experiment which performs an A/B test. Performs analysis against the A and B using the job metric provider |
| [Preview Stack Testing](examples/preview-testing) | Rollout which launches an experiment that tests a preview stack (which receives no production traffic) |
| [Canary with istio (1)](examples/istio) | Rollout which uses host-level traffic splitting during update |
| [Canary with istio (2)](examples/istio-subset) | Rollout which uses subset-level traffic splitting during update |

Before running an example:

1. Install Argo Rollouts

- See the document [Getting Started](https://argoproj.github.io/argo-rollouts/getting-started/)

2. Install Kubectl Plugin

- See the document [Kubectl Plugin](https://argoproj.github.io/argo-rollouts/features/kubectl-plugin/)

To run an example:

1. Apply the manifests of one of the examples:

```bash
kustomize build <EXAMPLE-DIR> | kubectl apply -f -
```

2. Watch the rollout or experiment using the argo rollouts kubectl plugin:

```bash
kubectl argo rollouts get rollout <ROLLOUT-NAME> --watch
kubectl argo rollouts get experiment <EXPERIMENT-NAME> --watch
```

3. For rollouts, trigger an update by setting the image of a new color to run:
```bash
kubectl argo rollouts set image <ROLLOUT-NAME> "*=ghcr.io/yourdevops/rollouts-demo:yellow"
```

## Images

Available image colors are: red, orange, yellow, green, blue, purple (e.g. `ghcr.io/yourdevops/rollouts-demo:yellow`). Also available are:
* High error rate images, prefixed with `bad` (e.g. `ghcr.io/yourdevops/rollouts-demo:bad-yellow`)
* High latency images, prefixed with `slow` (e.g. `ghcr.io/yourdevops/rollouts-demo:slow-yellow`)


## Releasing

Images are built and pushed automatically by GitHub Actions on push to `master`. See `.github/workflows/build-push.yaml`. All 18 color/error/latency variants are published to `ghcr.io/yourdevops/rollouts-demo`.
