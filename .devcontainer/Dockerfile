# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.245.2/containers/alpine/.devcontainer/base.Dockerfile

ARG VARIANT="3.16"
FROM mcr.microsoft.com/vscode/devcontainers/base:0-alpine-${VARIANT}

USER vscode
ENV PATH=$HOME/go/bin:$PATH

RUN sudo apk add go gopls apk-tools \
  && go install github.com/spf13/cobra-cli@latest