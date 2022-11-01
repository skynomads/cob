# cob

Lightning fast builds for [apko](https://github.com/chainguard-dev/apko) and [melange](https://github.com/chainguard-dev/melange).

* Run builds in parallel
* (Re)build changed files only
* Watch mode that triggers builds on-demand
* Runs custom commands before and after builds

Note: This is alpha software.

## Install

```sh
go install github.com/skynomads/cob@main
```

## Example

`cob.yaml`:

```yaml
package:
  source:
    - pkg/*.yaml
  target: dist/pkg
  post-build: |
    echo post-build
image:
  source:
    - img/*.yaml
  target: dist/img
  ref:
    hello: docker.io/example/hello:1.0.0
signing-key: melange.rsa
keyring-append:
  - melange.rsa.pub
```

To build, run `cob build`. To watch, run `cob watch`.