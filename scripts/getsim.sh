#! /bin/bash

set -e

cat /proc/self/cgroup | grep 'docker/' | tail -1 | sed 's/^.*\///' > /opt/dsengine/simid.dat
