# locksmith

locksmith is a reboot manager for the CoreOS update engine which uses
etcd to ensure that only a subset of a cluster of machines are rebooting
at any given time. `locksmithd` runs as a daemon on CoreOS machines and is
responsible for controlling the reboot behaviour after updates.

## Configuration

There are three different strategies that `locksmithd` can use after the update
engine has successfully applied an update:

- `etcd-lock` - reboot after first taking a lock in etcd.
- `reboot` - reboot immediately without taking a lock.
- `best-effort` - if etcd is running, then do `etcd-lock`; otherwise, `reboot`.

These strategies can be configured via `/etc/coreos/update.conf` with a line that looks like:

```
REBOOT_STRATEGY=reboot
```

The reboot strategy can also be configured through [cloud-config](https://github.com/coreos/coreos-cloudinit/blob/master/Documentation/cloud-config.md#update).

The default strategy is `best-effort`.

## Usage

`locksmithctl` is a simple client that can be use to introspect and control the
lock used by locksmith.  It is installed by default on CoreOS.

### Listing the Holders

```
$ locksmithctl status
Available: 0
Max: 1

MACHINE ID
69d27b356a94476da859461d3a3bc6fd
```

### Unlock Holders

In some cases a machine may go away permanently or semi-permanently while
holding a reboot lock. A system administrator can clear the lock of a specific
machine using the unlock command:

```
$ locksmithctl unlock 69d27b356a94476da859461d3a3bc6fd
```

### Maximum Semaphore

By default the reboot lock only allows a single holder. However, a user may
want more than a single machine to be upgrading at a time. This can be done by
increasing the semaphore count.

```
$ locksmithctl set-max 4
Old: 1
New: 4
```


## Implementation details

The following section describes how locksmith works under the hood.

### Semaphore

locksmith uses a [semaphore][semaphore] in etcd (located at the key
`coreos.com/updateengine/rebootlock/semaphore`) to coordinate the reboot lock.

The semaphore is a JSON document, describing a simple semaphore, that clients [swap][cas]
to take the lock.

When it is first created it will be initialized like so:

```json
{
	"semaphore": 1,
	"max": 1,
	"holders": []
}
```

For a client to take the lock, the document is swapped with this:

```json
{
	"semaphore": 0,
	"max": 1,
	"holders": [
		"69d27b356a94476da859461d3a3bc6fd"
	]
}
```

[semaphore]: http://en.wikipedia.org/wiki/Semaphore_(programming)
[cas]: https://github.com/coreos/etcd/blob/master/Documentation/api.md#atomic-compare-and-swap

## Consul backend support
Consul is a tool for discovering and configuring services in your infrastructure.

Consul Key/Value Store: Applications can make use of Consul's hierarchical key/value store for any number of purposes including: dynamic configuration, feature flagging, coordination, leader election, etc. The simple HTTP API makes it easy to use.

[consul]: http://www.consul.io/intro/index.html
[consul-api]: https://github.com/armon/consul-api

### Consul usage

In current commits only locksmithctl is suppported.

```
>>locksmithctl --backend=consul --consul-endpoint=consul-1:8500 status
{"semaphore":0,"max":1,"holders":["373aa9f8cfab4f80bf7ca3a272a85628"]}
Available: 0
Max: 1

MACHINE ID
373aa9f8cfab4f80bf7ca3a272a85628

locksmithctl addl flags:
	--backend=etcd	:etcd or consul as backend for locksmith. Defaults to the etcd.
	--consul-endpoint=127.0.0.1:8500	consul endpoint for locksmith. Defaults to the local instance.
```
