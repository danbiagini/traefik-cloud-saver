# Traefik Cloud Saver Plugin

A Traefik plugin, `traefik-cloud-saver` which saves cloud resource costs by automatically stopping instances that are not being used.

Think of it like "turning the lights off when the room is empty."

[![Build Status](https://github.com/danbiagini/traefik-cloud-saver/workflows/Main/badge.svg?branch=master)](https://github.com/danbiagini/traefik-cloud-saver/actions)

## Features

- Monitors traffic rates for Traefik services
- Automatically scales down cloud instances during low traffic periods
- Configurable traffic thresholds and monitoring windows
- Supports filtering specific routers to monitor (TODO)

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
## Logs

The plugin logs to traefik logs, search for `traefik-cloud-saver` in the logs.

## Plugins Catalog

Traefik plugins are stored and hosted as public GitHub repositories.

Every 30 minutes, the Plugins Catalog online service polls Github to find plugins and add them to its catalog.

If something goes wrong with the integration of your plugin, Plugins Catalog will create an issue inside your Github repository and will stop trying to add your repo until you close the issue.

## Troubleshooting

If Plugins Catalog fails to recognize your plugin, you will need to make one or more changes to your GitHub repository.

In order for your plugin to be successfully imported by Plugins Catalog, consult this checklist:

- The `traefik-plugin` topic must be set on your repository.
- There must be a `.traefik.yml` file at the root of your project describing your plugin, and it must have a valid `testData` property for testing purposes.
- There must be a valid `go.mod` file at the root of your project.
- Your plugin must be versioned with a git tag.
- If you have package dependencies, they must be vendored and added to your GitHub repository.

