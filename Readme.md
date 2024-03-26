# Presentation

This project is a Go server that can be used to demonstrate an implementation of **CUSTOM** option of the **AuthorizationPolicy**.
The tutorial is available [here](https://medium.com/@marc.guerrini/diy-istio-custom-authorization-policy-ecf1927e498a).

This Go module exposes the gRPC interface invoked by **Istio** to validate requests (relies on https://github.com/envoyproxy/go-control-plane/)

We start the gRPC listener on port 9000 (or whatever port injected in command line **-grpc=xxx**).

## How to build
Here is the command to create the Docker image used by the medium tutorial.

```shell
docker build -t mgu/authz-ext-basic .
```

## Request processing
For each request our auth server will check that the HTTP header **tested-header** is present.

If the header is missing, our auth server will reject the request.

If the header is present, the request will be forwarded to the target, and enriched with a new HTTP header **generated-header**.

