package main

import "github.com/ConSol-Monitoring/nagflux/pkg/nagflux"

// Build contains the current git commit id
// compile passing -ldflags "-X main.Build <build sha1>" to set the id.
var Build string

func main() {
	nagflux.Nagflux(Build)
}
