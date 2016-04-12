---
title: Autoscaling
menu_order: 40
---
An autoscaling configuration begins with a small fixed cluster:

* Configured as per the [Uniform Fixed Cluster](/site/operational-guide/uniform-fixed-cluster.md)
  scenario
* Hosted on reserved or protected instances to ensure long-term
  stability
* Ideally sized at a minimum of three or five nodes (you can make your
  fixed cluster bigger to accommodate base load as required; a minimum
  is specified here in the interests of resilience only)

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
