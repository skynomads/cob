# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.245.2/containers/alpine/.devcontainer/base.Dockerfile

ARG VARIANT="3.16"
FROM mcr.microsoft.com/vscode/devcontainers/base:0-alpine-${VARIANT}

USER vscode
ENV PATH="/home/vscode/go/bin:$PATH"
WORKDIR /home/vscode

RUN echo "@testing http://dl-cdn.alpinelinux.org/alpine/edge/testing" | sudo tee -a /etc/apk/repositories \
  && sudo apk add go build-base bubblewrap apk-tools apko@testing \
  && go install github.com/goreleaser/goreleaser@latest \
  && go install golang.org/x/tools/gopls@latest \
  && go install golang.org/x/tools/cmd/goimports@latest \
  && git clone https://github.com/chainguard-dev/melange.git \
  && cd melange && sudo make melange install && cd .. && rm -rf melange