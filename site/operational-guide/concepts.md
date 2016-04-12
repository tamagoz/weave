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

A peer is a running instance of Weave Net, typically one per host.

## Peer Name

Peers in the weave network are identified by a 48-bit value formatted
like an ethernet MAC address e.g. `01:23:45:67:89:ab`. This 'peer
name' is used for various purposes:

* Routing of packets between containers on the overlay network
* Recording the origin peer of DNS entries
* Recording ownership of IP address ranges

Whilst it is desirable for the peer name to remain stable across
restarts, it is essential that it is unique - if two or more peers
share the same name chaos will ensue, including but not limited to
double allocation of addresses and inability to route packets on the
overlay network. Consequently when the router is launched on a host it
derives its peer name in order of preference:

* From the command line; user is responsible for uniqueness and
  stability
* From the BIOS product UUID, which is generally stable across
  restarts and unique across different physical hardware and certain
  cloned VMs
* From the hypervisor UUID, which is generally stable across restarts
  and unique across VMs which do not provide access to a BIOS product
  UUID
* From a random value, practically unique across different physical
  hardware and cloned VMs but not stable across restarts

The best strategy for assigning peer names depends on the type and
method of your particular deployment and is discussed in more detail
below.

## Peer Discovery

Peer discovery is a mechanism which allows peers to learn about new
weave hosts from existing peers without being explicitly told. Peer
discovery is enabled by default.

## Network Partition

A network partition is a transient condition whereby some arbitrary
subsets of peers are unable to communicate with each other for the
duration - perhaps because a network switch has failed, or a fibre
optic line severed. Weave is designed to allow peers and their
containers to make maximum safe progress under conditions of
partition, healing automatically once the partition is over.

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

Certain information is remembered between launches of weave (for
example across reboots):

* The division of the IP allocation range amongst peers
* Allocation of addresses to containers

The persistence of this information is managed transparently in a
volume container but can be destroyed explicitly if necessary.