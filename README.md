# CronHorizontalPodAutoscaler Controller

A Kubernetes custom controller for managing `CronHorizontalPodAutoscaler` (CronHPA) Custom Resource Definitions (CRDs). This controller allows you to schedule automatic scaling of HorizontalPodAutoscalers (HPAs), Deployments, or StatefulSets based on cron expressions, supporting timezones and one-time executions.

Built with [Kubebuilder](https://book.kubebuilder.io/), using [gocron](https://github.com/go-co-op/gocron) for scheduling and [Uber FX](https://github.com/uber-go/fx) for dependency injection.

## Features

- **Scheduled Scaling**: Define jobs with cron schedules to update minReplicas on HPAs or replicas on Deployments/StatefulSets.
- **Timezone Support**: Schedules are interpreted in specified timezones (defaults to "Asia/Bangkok"), converted internally to UTC.
- **Run Once Option**: Jobs can be set to run only once and self-remove after execution.
- **Target Flexibility**: Scales HPAs (minReplicas), Deployments, or StatefulSets (replicas) via unstructured updates.
- **Logging and Verification**: Detailed info/debug logs for reconciliation, job registration, and scaling actions, including post-update verification.
- **Finalizer Handling**: Ensures cleanup of scheduled jobs on CR deletion.

## Prerequisites

- Kubernetes cluster (v1.20+ recommended)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [Go](https://golang.org/dl/) (v1.22+)
- [Kubebuilder](https://book.kubebuilder.io/quick-start.html) (v3.15+ for development)

## Installation

### Using Manifests

Apply the generated manifests from Kubebuilder:

```sh
kubectl apply -k config/manager
```

## Usage

### Create a CronHPA Resource

Apply a YAML manifest like the following:

```yaml
apiVersion: autoscaling.cloudnatician.com/v1
kind: CronHorizontalPodAutoscaler
metadata:
  name: cronhpa-sample
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: autoscaling/v2
    kind: HorizontalPodAutoscaler
    name: hpa-sample
  jobs:
    - name: "scale-up-9am"
      schedule: "0 0 9 * * *"
      timezone: "Asia/Bangkok"
      minReplicas: 20
      runOnce: false
    - name: "scale-down-5pm"
      schedule: "0 0 17 * * *"
      timezone: "Asia/Bangkok"
      minReplicas: 3
      runOnce: false
```

- `scaleTargetRef`: References the target HPA, Deployment, or StatefulSet.
- `jobs`: List of scaling jobs with cron schedule, optional timezone, minReplicas, and runOnce flag.

### Monitoring

Check controller logs for reconciliation and scaling events:

```sh
kubectl logs -f deployment/cronhpa-controller-manager -c manager
```

Example log output during scaling:

```json
INFO    Executing scheduled job {"cr": "cronhpa-sample", "job": "scale-up-9am", "target": "hpa-sample", "minReplicas": 20}
INFO    Updating resource {"kind": "HorizontalPodAutoscaler", "name": "hpa-sample", "namespace": "default", "replicas": 20}
INFO    Current minReplicas {"current": 3}
INFO    HPA minReplicas updated successfully {"from": 3, "to": 20}
```

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/my-feature`).
3. Commit your changes (`git commit -am 'Add my feature'`).
4. Push to the branch (`git push origin feature/my-feature`).
5. Open a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

For questions or issues, open a GitHub issue.