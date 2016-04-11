---
title: Concepts
menu_order: 10
---
This section describes some essential concepts with which you will
need to be familiar before moving on to the example deployment
scenarios.

> Author's Note: I have deliberately excluded any mention of
> weave command syntax in this section

## Peer

A peer is a host on which you have installed weave.

## Peer Name

Peers in the weave network are identified by a 48-bit value formatted
like an ethernet MAC address e.g. `01:23:45:67:89:ab`. This 'peer
name' is used for various purposes:

* Routing of packets between containers on the overlay network
* Recording the origin peer of DNS entries
* Recording ownership of IP address ranges

Whilst the first two uses tolerate the peer name changing on router
restarts, the third use means that address space can be 'lost' (e.g.
recorded as being owned by a now non-existent peer name) and so it is
desirable for the peer name to be as stable as possible across
restarts. Consequently when the router is launched on a host it
derives its peer name in order of preference:

* From the command line; specifier is responsible for uniqueness and
  stability
* From the BIOS product UUID, which is generally stable across restarts
  and unique across different physical hardware and cloned VMs
* From a random value, unique across different physical hardware and
  cloned VMs but not stable across restarts

More important still than the stability constraint is uniqueness;
address space lost due to a peer name change can be recovered with an
administrative action, but if two or more peers share the same name
chaos will ensue, including but not limited to double allocation of
addresses and inability to route packets on the overlay network.

The best strategy for assigning peer names depends on the type and
method of your particular deployment and is discussed in more detail
below.

## Peer Discovery

Peer discovery is a mechanism which allows peers to learn about new
weave hosts from existing peers without being explicitly told. Peer
discovery is enabled by default.

## Network Partition

A network partition is a transient condition whereby some arbitrary
subsets of peers are unable to communicate with each other. Weave is
designed to allow peers and their containers to make maximum safe
progress under conditions of partition, healing automatically once the
partition is over.

## IP Address Manager (IPAM)

IPAM is the subsystem responsible for dividing up a large contiguous
block of IP addresses (known as the IP allocation range) amongst peers
so that individual addresses may be uniquely assigned to containers
anywhere on the overlay network.

When a new network is formed an initial division of the IP allocation
range must be made. Two (mutually exclusive) mechanisms with different
tradeoffs are provided to perform this task: seeding and consensus.

## Seeding

Seeding requires each peer to be told the list of peer names amongst
which the address space is to be divided initially. There are some
constraints and consequences:

* Every peer added to the network _must_ receive the same seed list,
  for all time, or they will not be able to join together to form a
  single cohesive whole
* Because the 'product UUID' and 'random value' methods of peer name
  assignment are unpredictable, the end user must by necessity also
  specify peer names
* Even though every peer _must_ receive the same seed, that seed does
  _not_ have to include every peer in the network, nor does it have to
  be updated when new peers are added (in fact due to the first
  constraint above it may not be)

Example configurations are given in the section on deployment
scenarios.

## Consensus

Alternatively, when a new network is formed for the first time peers
can be configured to co-ordinate amongst themselves to automatically
divide up the IP allocation range - this process is known as
consensus, and is implemented via a leader election algorithm.
Consensus based bootstrapping requires each peer to be told the total
number of expected peers (the 'initial peer count') in order to avoid
small independent groups of peers from electing different leaders
under conditions of partition.

Example configurations are given in the section on deployment
scenarios.

## Electors and Observers

When using consensus-based configuration then by default every peer
included in the network is a candidate for leadership. This does not
always make sense (for example when adding relatively short lived
peers to an existing cluster in response to a scale-out event) and so
an option is provided to start a peer as an _observer_. Such peers
will not put themselves forward as leaders, but will abide by the
results of an election amongst elector-peers.

Example configurations are given in the section on deployment
scenarios.

## Persistence

If you relaunch weave with _exactly_ the same parameters as the
previous invocation (as will happen if it is restarted automatically
by Docker), certain information is remembered:

* Runtime modifications to the target peer list enacted via the
  command line
* The division of the IP allocation range amongst peers
* Allocation of addresses to containers

The persistence of this information is managed transparently inside
the router container but can be destroyed explicitly if necessary.
