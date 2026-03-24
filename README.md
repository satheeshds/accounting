## Accounting Service

Make targets are provided for common developer workflows:

- `make build` – build the Go binary.
- `make test` – run the Go test suite.
- `make run` – run locally (writes the DB to `./data/accounting.db` by default; listens on the port specified by `APP_PORT` (default `8080`), which the Makefile passes to the service as `PORT`).
- `make docker-build` – build the Docker image (defaults to `accounting:latest`).
- `make docker-run` – builds (via the `docker-build` dependency) and runs the image locally, mapping host port `HOST_PORT` (default `8080`) to container port `CONTAINER_PORT` (default `80`).
- `make docker-push` – push the tagged image.

Export `AUTH_USER` and `AUTH_PASS` in your shell (or prefix the command) before invoking `make docker-run`; these credentials are required for the container to start.

Use `make help` to list all available targets.
