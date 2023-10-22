# Tiny Systems All-In-One module
Collection of various components combined into the first module.

### Development run
Switch kubectl to the proper context.

```shell
go run cmd/main.go run --version 0.1.0 --name my-module
```
### Build & Release

```shell
go run cmd/main.go build --version 0.1.0 --name main --devkey devkeyabc1111
```

### Prerequisites 
* Golang v1.20+
* Docker
* Kubernetes  1.26+

