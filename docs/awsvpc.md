# AWS VPC

This document contains information on AWS VPC based underlay network
implementation for Weave.

The underlay network uses AWS VPC routing facilities to route packets among
containers instead of relying on the pcap or fastdp implementations.

## Limitations

* By default, Amazon VPC has an upper limit equal to 50 routes within a single route
  table.
* All weave peers (incl. containers) should reside within the same VPC.
* Encrypted communication is not supported.

## Setup

* Follow [Flannel AWS VPC](https://coreos.com/blog/introducing-flannel-0.5.0-with-aws-and-gce/) instructions for setting up your Amazon AWS VPC.
  Open required ports for weave nodes communication (TODO: list them).
* Download `weave` script from [here](https://raw.githubusercontent.com/brb/weave/underlay-net/weave).
* Launch Weave on AWS EC2 hosts by executing (on hosts):

```bash
sudo WEAVE_NO_FASTDP=1 DOCKERHUB_USER=brb0 weave launch --ipalloc-range=10.20.0.0/16 --awsvpc --aws-routetableid=ROUTE_TABLE_ID PEER_IP
```
  where `ROUTE_TABLE_ID` corresponds to the id of the VPC route table (e.g. `rtb-259b9740`).
* Assign an IP address to the weave bridge on each host by executing `sudo DOCKERHUB_USER=brb0 weave expose`
* Enable ARP proxying on the weave bridge by issuing on each host: `echo 1 > /proc/sys/net/ipv4/conf/$BRIDGE/proxy_arp`
* Delete the weave route on each host: `ip route del 10.20.0.0/16 dev weave`
* Start containers.

## Implementation details

First of all, we have extended the `Allocator` struct by adding a field for newly
introduced `Monitor` interface. The interface requires to implement a single
`HandleUpdate(old, new []address.Range)` function which gets executed whenever there are any changes in
an address ring. This happens when:

* The ring has been established.
* A peer has donated some of address range it owned before.

The `old` parameter corresponds to an address range which has been owned before
donation and the `new` - to the current address range.

Upon the donation, the function is executed on related peers. E.g., if peer A that has previously owned the 10.0.0.0-10.0.0.15 range donates
the first half to peer B that owns range 10.0.1.0-10.0.1.15, then the function is executed in particular order by:

* Peer A with `old=["10.0.0.0-10.0.0.15"]`, `new=["10.0.0.8-10.0.0.15"]`.
* Peer B with `old=["10.0.1.0-10.0.1.15"]`, `new=["10.0.0.8-10.0.0.15",
  "10.0.1.0-10.0.1.15"]`.

Next, the `HandleUpdate` function of the AWS VPC monitor is responsible for
managing the AWS VPC route table. Before applying any change, it breaks the
passed ranges into CIDR aligned ranges. In addition, the function modifies
system route tables on the host machine.

Finally, when using the AWS VPC monitor, we set `isCIDRAligned` to true which
means that peers donate only CIDR aligned ranges.

## TODO

* Store address ranges in a Radix tree to be able to efficiently minimize the number of required entries
  in the AWS VPC route table due to CIDR alignments.
* Do CIDR aligned allocations when establishing the ring.
* Remove unnecessary grouping of address ranges (`groupRanges` in
  `HandleUpdated`), because AWS VPC allows us to insert a route if it has
  already existed.
* `ring.OwnedRanges` seems to return ranges which should have been merged. Fix
  it if it does not introduce unwanted behaviour.
* Extend `weave` script for setting up `proxy_arp`, removing default routes,
  etc.
* Cleanup routes upon termination.

## Author

[Martynas Pumputis](mailto:martynasp@gmail.com)
