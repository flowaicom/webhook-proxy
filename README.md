# Flow-judge webhook proxy

This project is a proxy enabling communication between the Flow Judge client and Baseten webhooks 
for async inference. It streams webhook payloads to the client without the client needing
to expose an endpoint to the Internet. 

<a href="https://github.com/flowaicom/webhook-proxy/actions/workflows/release.yml" target="_blank">
    <img src="https://github.com/flowaicom/webhook-proxy/actions/workflows/release.yml/badge.svg" alt="Build">
</a>

## Features

- **Webhook Proxy**: Acts as a proxy for incoming webhooks from Baseten async.
- **HTTP Streaming**: Exposes an endpoint for HTTP streaming that the [flow-judge](https://github.com/flowaicom/flow-judge) Python client can connect to.

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/flowaicom/webhook-proxy.git
   cd webhook-proxy
   ```
2. Build the project:
   ```bash
   go build -o proxy .
   ```
   or
   ```bash
   make
   ```

3. Run the proxy:
   ```bash
   ./proxy -addr 0.0.0.0:8000
   ```

### Running with Docker

```
docker pull ghcr.io/flowaicom/webhook-proxy:latest
docker run --name=flowai-proxy -d -p 8000:8000 ghcr.io/flowaicom/webhook-proxy:latest
```

## Configuration

Proxy can be configured at the startup with the following flags.

| Flag                      | Default value  | Description                                                                                                                                                                                                           |
|---------------------------|----------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `-addr`                   | `0.0.0.0:8000` | The interface and port which the proxy should listen on.                                                                                                                                                              |
| `-timeout`                | 120            | Timeout in seconds after which the client connection will be dropped. Webhooks delivered and not sent to clients within this timeframe will also be dropped.                                                          |
| `-metrics-token`          | -              | Bearer token for accessing `/metrics` endpoint serving Prometheus metrics. Takes precendence over the environment variable.                                                                                           |
| `PROXY_METRICS_TOKEN`     | -              | Alternative way (env variable) of configuring the token setting above.                                                                                                                                                |
| `-allow-insecure-metrics` | `false`        | Whether to allow access to `/metrics` endpoint without authentication. If set to `false`(default) and token not set with the options above, the program will generate random token and print it to stdout at startup. 

## API

For detailed API description see [`docs/`](https://github.com/flowaicom/webhook-proxy/tree/main/docs)
directory.

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature/YourFeature`).
3. Make your changes and commit (`git commit -m 'Add some feature'`).
4. Push to the branch (`git push origin feature/YourFeature`).
5. Create a new Pull Request.
