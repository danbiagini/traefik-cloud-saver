# Traefik Cloud Saver Plugin

A Traefik plugin, `traefik-cloud-saver` which saves cloud resource costs by automatically stopping instances that are not being used.

Think of it like "turning the lights off when the room is empty."

![Build Status](https://github.com/danbiagini/traefik-cloud-saver/actions/workflows/main.yml/badge.svg?branch=master)

## Features

- Monitors traffic rates for Traefik services
- Automatically shuts down cloud instances during low traffic periods
- Configurable traffic thresholds and monitoring windows
- Supports filtering specific routers to monitor (TODO)

### Supported Clouds

- Google Cloud Platform
- Mock (for testing)

## Usage

Currently the plugin is not available in the [Traefik Plugin Catalog](https://plugins.traefik.io/), so you need to build it yourself.  But it's easy to do, for a sample container build see the 'build-test-container' target in the [Makefile](Makefile).

I hope to get the plugin added to the Traefik Plugin Catalog soon.

### Configuration


```yaml
# Static configuration

experimental:
  localPlugins:
    traefik_cloud_saver:
      moduleName: github.com/danbiagini/traefik-cloud-saver
      version: v0.1.0

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
#### Local Mode

This plugin can be run in local mode, it requires a specific filesystem structure in the container.  See the [Makefile](Makefile) 'build-test-container' target and [Docker Compose](test/docker-compose.yml) for an example.

## Troubleshooting

Enable debug logging to get more information.  See the [Configuration](#configuration) section for details.

### Logs
The plugin logs to traefik logs, search for `traefik-cloud-saver` in the logs.

### Integration Validator
There is an integration test that can be run to check that the GCE credentials are valid.  See the file [compute_integration_test.go](test/compute_integration_test.go) for details on how to run it.