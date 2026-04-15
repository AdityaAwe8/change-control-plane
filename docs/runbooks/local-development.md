# Local Development Runbook

## Start Dependencies

```bash
make compose-up
```

## Start the API

```bash
make migrate
make run-api
```

## Start the Worker

```bash
make run-worker
```

## Start the Web App

```bash
make web-install
make web-dev
```

## Verify the Baseline

```bash
make test
make smoke
```
