package collector

import "github.com/ConSol-Monitoring/nagflux/pkg/data"

type ResultQueues map[data.Target]chan Printable
