# Traefik Cloud Saver Plugin

<p align="center">
  <img src=".assets/logo.png" alt="Traefik Cloud Saver" width="200"/>
  <br>
  <em>Automatically scale down idle cloud resources to reduce costs</em>
</p>

💸 Think of it as "turning the lights off when the room is empty." 💸

[![Build Status](https://github.com/danbiagini/traefik-cloud-saver/actions/workflows/main.yml/badge.svg?branch=master)](https://github.com/danbiagini/traefik-cloud-saver/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/danbiagini/traefik-cloud-saver)](https://goreportcard.com/report/github.com/danbiagini/traefik-cloud-saver)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## 🚀 Features

- 📊 Real-time traffic monitoring for Traefik services
- 💤 Automatic instance shutdown during low traffic periods
- ⚙️ Configurable thresholds and monitoring windows
- 🔜 Service-specific monitoring with router filtering (coming soon)
- 📝 Detailed debug logging

## 🌩️Supported Clouds

- ✅ Google Cloud Platform (GCP)
- 🧪 Mock Provider (for testing)
- 🔜 AWS (coming soon)
- 🔜 Azure (coming soon)

## 🔧 Quick Start

The plugin is not available in the [Traefik Plugin Catalog](https://plugins.traefik.io/) *yet*, so you need to build it yourself.  But it's easy to do.

1. **Install the plugin**
See the sample container build in the 'build-test-container' target in the [Makefile](Makefile).

2. **Configure the plugin**
```yaml
# Static configuration

experimental:
  localPlugins:
    traefik_cloud_saver:
      moduleName: github.com/danbiagini/traefik-cloud-saver
      version: v0.1.0
```

```yaml
providers:
  plugin:
    traefik_cloud_saver:
      windowSize: 1m
      metricsURL: http://localhost:8080/metrics
      apiURL: http://localhost:8080/api
      trafficThreshold: 1
      debug: true
      cloudConfig:
        type: gcp
        region: <your-region>
        zone: <your-zone>
        credentials:
          secret: <path-to-service-account-json-file>
          type: service_account
```

You need to provide a service account json file in the container, for example at `/etc/gcp/test_service_account.json`, or use a different path and change the `secret` path in the above config.

## 🔍 How It Works

1. **Traffic Monitoring**: Continuously monitors request rates through Traefik's metrics
2. **Threshold Analysis**: Compares traffic against configured thresholds
3. **Scale Decision**: Triggers scale-down when traffic drops below threshold
4. **Cloud Integration**: Executes scaling through cloud provider APIs


## 🐛 Troubleshooting

### Common Issues

1. **Plugin Not Loading**
   - Verify plugin configuration in Traefik
   - Check logs for initialization errors

2. **Scaling Not Working**
   - Confirm cloud credentials are valid
   - Check traffic thresholds
   - Enable debug logging

### Debug Logging

Enable debug logging in configuration:
```yaml
debug: true
```

### Logs
The plugin logs to traefik logs, search for `traefik-cloud-saver` in the logs.

### Integration Validator
There is an integration test that can be run to check that the GCE credentials are valid.  See the file [compute_integration_test.go](test/compute_integration_test.go) for details on how to run it.

## 🛠️ Development

### Prerequisites

- Go 1.x
- Docker
- Access to a cloud provider account

### Building

```bash
# Install dependencies
make vendor

# Run tests
make test

# Build test container
make build-test-container
```

## 🔬 Testing

### Unit Tests
```bash
make test
```

### Integration Tests
```bash
# Set up GCP credentials first
make integration-test


## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📜 License

This project is licensed under the Apache License, Version 2.0 - see the [LICENSE](LICENSE) file for details.

```
Copyright 2024 Dan Biagini

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

## 🙏 Acknowledgments

- Traefik team for their amazing reverse proxy
- Contributors and users of this plugin

