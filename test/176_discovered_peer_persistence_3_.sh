#! /bin/bash

. ./config.sh

start_suite "Check resume uses persisted discovered peers"

# Create a chain
weave_on $HOST1 launch --ipalloc-init consensus=3
weave_on $HOST2 launch --ipalloc-init consensus=3 $HOST1
weave_on $HOST3 launch --ipalloc-init consensus=3 $HOST2

# Wait for discovery
weave_on $HOST1 prime

# Stop them all
weave_on $HOST1 stop
weave_on $HOST2 stop
weave_on $HOST3 stop

# Resume first and last nodes in the chain
weave_on $HOST1 launch --resume
weave_on $HOST3 launch --resume

# Ensure they're connected
start_container $HOST1 --name c1
start_container $HOST3 --name c3
assert_raises "exec on $HOST1 c1 $PING $(container_ip $HOST3 c3)"

end_suite
