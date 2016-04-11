---
title: Administrative Tasks
menu_order: 60
---
## Configuring Weave to Start Automatically on Boot

`weave launch` runs all weave's containers with a Docker restart
policy of `always`, so as long as you have launched weave manually
once and your system is configured to start Docker on boot then weave
will be started automatically on system restarts.

If you're aiming for a non-interactive installation, you can use
systemd to launch weave after Docker - see [systemd docs](TODO) for
details.

> Author's Note: I'm going to pull the complete systemd doc into this
> section

## Detecting Lost IPAM Space

The recommended way of removing a peer is to run `weave reset` on that
peer before the underlying host is decommissioned or repurposed - this
ensures that the portion of the IPAM allocation range assigned to the
peer is released for reuse. Under certain circumstances this operation
may not be successful, or indeed possible:

* If the peer in question is partitioned from the rest of the network
  when `weave reset` is executed on it
* If the underlying host is no longer available to execute `weave
  reset` - for example due to a hardware failure or other unplanned
  termination

In some cases you may already be aware of the problem, as you were
unable to execute `weave reset` successfully or because you know
through other channels that the host has died - in these cases you can
proceed straight to the section on reclaiming lost space.

However in some scenarios (for example when using autoscaling and
spot-instances that can be destroyed without notice) it may not be
obvious that space has been lost, in which case you can check for it
periodically with the following command on any peer:

    weave status ipam

This will list the names of unreachable peers; if you are satisifed
that they are truly gone, rather than temporarily unreachable due to a
partition, you can reclaim their space by following the advice in the
next section.

## Reclaiming Lost IPAM Space

When a peer dies unexpectedly the remaining peers will consider its
address space to be unavailable even after it has remained unreachable
for prolonged periods; there is no universally applicable time limit
after which one of the remaining peers could decide unilaterally that
it is safe to appropriate the space for itself, and so an
adminstrative action is required to reclaim it.

The `weave rmpeer` command is provided to perform this task, and must
be executed on one of the remaining peers. That peer will take
ownership of the freed adress space.

## Rolling Upgrades

Protocol versioning and feature negotation are employed in Weave Net
to enable incremental rolling upgrades - each release maintains the
ability to speak to the previous version (at a minimum), and connected
peers only utilise features which both support. The general upgrade
procedure is as follows:

On each peer in turn:

* Download the new weave script to a temporary location e.g.
  `/path/to/new/weave`
* Pull the new images with `/path/to/new/weave setup`
* Stop the old weave with `weave stop` (or `systemctl weave stop` if
  you're using a systemd unit file)
* Replace the existing script with the new one
* Start the new weave with `weave launch <existing peer list>` (or
  `systemctl weave start` if you're using a systemd unit file)

Always check the release notes for specific versions in case there are
any special caveats or deviations from the standard procedure.

> Author's Note:
> * `weave setup` with the new script before we stop the old router
>   minimises downtime
