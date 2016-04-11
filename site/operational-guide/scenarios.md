---
title: Deployment Scenarios
menu_order: 20
---
This section discusses various deployment scenarios, documenting the
best-practice operational guidance for each use case.

## Manual/Incremental

This scenario is intended for evaluation, not production use. The
resulting weave network will survive host reboots without the use of a
systemd unit as long as Docker is configured to start on boot.

### Bootstrapping

On initial peer:

    weave launch
    weave consense

> Author's Note: the consense step here is not strictly necessary, but
> I include it so that a uniform pattern of launch + consense is
> ingrained as it _is_ important when extending the network

### Adding a Peer

On new peer:

    weave launch <existing peer list>
    weave consense

The consense step is required to ensure that the new peer has joined
to the existing network; you _must_ wait for this to complete
successfully before moving on to add further new peers.

> Author's Note: the point of serialising on consense after each
> additional peer is to prevent the problem whereby in this scenario
>
>     host1$ weave launch             # quorum = 1
>     host2$ weave launch host1       # quorum = 2
>     host3$ weave launch host1 host2 # quorum = 2
>
> host1 and host2/host3 can form independent cliques if host1 is
> partitioned from host2 and host3. An alternative (assuming #1721 is
> fixed) would be to mandate `weave connect` on existing peers as each
> new peer is added.

Then optionally on each existing peer:

    weave connect <new peer>

This step is not mandatory, but improves the robustness of network
reformation in the face of node failure as _nodes do not remember to
connect to discovered peers on restart_. (For example, if you did the
following:

    host-A$ weave launch; weave consense
    host-B$ weave launch; weave connect host-A; weave consense
    host-C$ weave launch; weave connect host-B; weave consense

host-C's ability to connect to host-A would be entirely dependent on
gossip discovery via host-B; if the whole network rebooted and host-B
did not come back up then host-A and host-C would not connect to each
other.)

### Stopping a Peer

A peer can be stopped temporarily, leaving its persisted data intact:

    weave stop

### Removing a Peer

On peer to be removed:

    weave reset

Then optionally on each remaining peer:

    weave forget <removed peer>

This step is not mandatory, but it will eliminate log noise and
spurious network traffic by stopping reconnection attempts and
preventing further connection attempts after restart.

## Uniform Fixed Cluster

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

## Autoscaling

An autoscaling configuration begins with a small fixed cluster:

* Configured as per the Uniform Fixed Cluster scenario
* Hosted on reserved or protected instances to ensure long-term
  stability
* Ideally sized at a minimum of three or five nodes (NB you can make
  your fixed cluster bigger to accomodate base load as required; we're
  only specifying a minimum here in the interests of resilience)

Building on this foundation, arbitrary numbers of dynamic peers can be
added or removed concurrently as desired, without requiring any
changes to the configuration of the fixed cluster.

### Scale-out

On additional peer:

    weave launch --observer <fixed cluster peers>

> Author's Note: `--observer` prevents a clique forming amongst a
> group of dynamically added peers if they become partitioned from the
> fixed cluster after having learned about each other. We only specify
> the fixed cluster peers as targets to avoid the machinery which is
> responding to the scale-out event from having to keep track of
> dynamic peers

### Scale-in

On peer to be removed:

    weave reset

## Uniform Dynamic Cluster

> Author's Note: TBD - this will talk about `--ipam-seed` and how it
> can be used with e.g. puppet to a) stand up an initial cluster and
> b) add arbitrary new nodes without requiring external
> synchronisation
