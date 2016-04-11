---
title: Manual/Incremental Deployment
menu_order: 20
---
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
