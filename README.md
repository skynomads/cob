# cob

Lightning fast builds for [apko](https://github.com/chainguard-dev/apko) and [melange](https://github.com/chainguard-dev/melange).

* Detects changes and rebuilds what changed
* Watch mode that triggers image rebuilds when packages change
* Run pre & post build commands

Note: This is alpha software.

## Example:

`cob.yaml`:

```yaml
package:
  source:
    - pkg/*.yaml
  target: dist/pkg
  pre-build: |
    echo pre-build
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

To build, run `cob build`. To watch, run `cob dev`.