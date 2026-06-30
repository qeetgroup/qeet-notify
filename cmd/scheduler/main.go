package main

import (
	"log"
)

// scheduler re-dispatches delayed workflow runs when their resume_at has passed.
// Currently the delay-ticker logic lives in domains/providers/sms/worker.go (DelayTicker).
// This binary will be the canonical home once the scheduler is extracted.
func main() {
	log.Fatal("scheduler: not yet extracted from sms worker — coming in a future slice")
}
