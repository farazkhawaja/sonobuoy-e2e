# Stage 1: Build environment to install Ginkgo and any dependencies
FROM golang:1.23-bullseye AS e2e-ginkgo

# Set GOBIN to install Ginkgo binary into /bin
ENV GOBIN=/bin

# Install Ginkgo for running tests
RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.11.0

# Create a working directory
WORKDIR /workspace

# Copy go.mod and go.sum first to cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the project files
COPY ./tests /workspace/tests

# Stage 2: Setup for running tests using Debian as the base image
FROM debian:bullseye AS e2e-tests

# Install necessary utilities like tar and gzip
RUN apt-get update && apt-get install -y tar gzip ca-certificates

# Copy Go binary, Ginkgo binary, and the project files from the first stage
COPY --from=e2e-ginkgo /usr/local/go /usr/local/go
COPY --from=e2e-ginkgo /bin/ginkgo /bin/ginkgo
COPY --from=e2e-ginkgo /workspace /workspace

# Set up the Go environment
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOROOT="/usr/local/go"

# Copy the run.sh script into the image
COPY run.sh /run.sh

# Ensure run.sh is executable
RUN chmod +x /run.sh

# Set the entrypoint to run.sh
ENTRYPOINT ["/run.sh"]
