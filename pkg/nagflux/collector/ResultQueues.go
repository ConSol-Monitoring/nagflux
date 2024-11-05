package collector

import "pkg/nagflux/data"

type ResultQueues map[data.Target]chan Printable
