package collector

import (
	"context"
	"fmt"
	"log"
	"sync"

	"golang.org/x/time/rate"
)

// ProjectPool manages concurrent collection of multiple GCP projects
// with rate limiting and error handling
type ProjectPool struct {
	projects    []string
	semaphore   chan struct{}
	rateLimiter *rate.Limiter
	errors      map[string]error
	mu          sync.Mutex
}

// NewProjectPool creates a new ProjectPool with the specified rate limits
//
// Parameters:
//   - projects: list of GCP project IDs to collect
//   - rps: requests per second limit (e.g., 10.0 for 10 requests/second)
//   - maxConcurrent: maximum number of concurrent project collections
//
// The rate limiter uses a burst size of rps*2 to allow for small bursts
// while maintaining the average rate over time.
func NewProjectPool(projects []string, rps float64, maxConcurrent int) *ProjectPool {
	return &ProjectPool{
		projects:    projects,
		semaphore:   make(chan struct{}, maxConcurrent),
		rateLimiter: rate.NewLimiter(rate.Limit(rps), int(rps*2)),
		errors:      make(map[string]error),
	}
}

// CollectAll collects resources from all projects concurrently with rate limiting.
//
// The method:
//   - Launches one goroutine per project
//   - Enforces concurrent collection limit via semaphore
//   - Applies rate limiting before each collection
//   - Collects errors per project without stopping other collections
//   - Returns an error if any project failed, but all projects are attempted
//
// This approach allows partial success: if 2 out of 40 projects fail,
// the other 38 will still be collected successfully.
func (p *ProjectPool) CollectAll(ctx context.Context, collector *Collector) error {
	var wg sync.WaitGroup

	for _, projectID := range p.projects {
		wg.Add(1)

		go func(pid string) {
			defer wg.Done()

			// Acquire semaphore to limit concurrent operations
			// This blocks if maxConcurrent projects are already being collected
			p.semaphore <- struct{}{}
			defer func() { <-p.semaphore }()

			// Apply rate limiting before making API calls
			// This respects GCP API quotas and prevents throttling
			if err := p.rateLimiter.Wait(ctx); err != nil {
				p.mu.Lock()
				p.errors[pid] = fmt.Errorf("rate limiter error: %w", err)
				p.mu.Unlock()
				log.Printf("Failed to acquire rate limit for project %s: %v", pid, err)
				return
			}

			// Collect project resources
			if err := collector.CollectProject(ctx, pid); err != nil {
				p.mu.Lock()
				p.errors[pid] = err
				p.mu.Unlock()
				// Log error but continue with other projects (partial failure handling)
				log.Printf("Failed to collect project %s: %v", pid, err)
			}
		}(projectID)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Return error if any projects failed, but include count for visibility
	if len(p.errors) > 0 {
		return fmt.Errorf("failed to collect %d projects", len(p.errors))
	}
	return nil
}

// Errors returns a copy of the errors map for inspection
// This allows callers to see which specific projects failed
func (p *ProjectPool) Errors() map[string]error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Return a copy to prevent concurrent modification
	errorsCopy := make(map[string]error, len(p.errors))
	for k, v := range p.errors {
		errorsCopy[k] = v
	}
	return errorsCopy
}
