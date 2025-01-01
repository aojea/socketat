# socketat

Short history first, Golang and linux namespaces doesn't mix well, this blog series explain in detail the problem:

* https://www.weave.works/blog/linux-namespaces-and-go-don-t-mix
* https://www.weave.works/blog/linux-namespaces-golang-followup

On containerized environments,like Kubernetes, this present a big problem to develop network applications
that may spawn multiple namespaces.

This library uses the technique described as "socketat" described in the [kernel mailing list](https://lore.kernel.org/patchwork/patch/217025/)

It basically enters the namespace to create the socket and returns the socket file descriptor.

That file descriptor any any operations on the sockets created are confined to the namespace,
but this time the user is not constrained by the golang limitations described.

The library wraps the net.Dial and net.Listen functions so they can run inside a network namespace:

```go
func DialAt(network, address string, ns int) (conn net.Conn, err error)

func ListenAt(network, address string, ns int) (net.Listener, error) {
```

### References:

Some good libraries to work with golang and linux namespaces:

1. https://github.com/containernetworking/plugins/blob/master/pkg/ns/ns_linux.go
2. https://github.com/vishvananda/netns
