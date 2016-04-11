---
title: Uniform Fixed Clusters
menu_order: 30
---
> Author's Note: I'm going to change this section to recommend systemd
> at which point the mentions of `weave launch` and `weave connect`
> will be replaced and I will talk about amending the PEER list in
> /etc/sysconfig/weave instead

A uniform fixed cluster has the following characteristics:

* Each node has identical configuration
* There is a controlled process for adding or removing nodes. The end
  user is responsible for ensuring that only one instance of the
  process is in-flight at a time

### Bootstrapping

This scenario describes a production deployment of a fixed number of
N nodes (N=1 in the degenerate case).

On each peer:

    weave launch <initial peer list>

The initial peer list may contain the address of the peer to which it
is being supplied, so the configuration may be identical for each
peer.

Then on _any_ peer:

    weave consense

To force consensus - future IP address allocations can now continue
under partition and it is safe to add new peers.

### Adding a Peer

On new peer:

    weave launch <existing peer list>
    weave consense

On each existing peer:

    weave connect <new peer>

### Removing a Peer

On peer to be removed:

    weave reset

On each remaining peer:

    weave forget <removed peer>
