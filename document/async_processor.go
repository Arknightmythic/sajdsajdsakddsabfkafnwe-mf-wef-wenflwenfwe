package document

import (
	"context"
	"dokuprime-be/external"
	"fmt"
	"log"
	"sync"
)

type ExtractionJob struct {
	DetailID int
	Request  external.ExtractRequest
}

type AsyncProcessor struct {
	externalClient *external.Client
	jobQueue       chan ExtractionJob
	wg             sync.WaitGroup
	workerCount    int
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	isShuttingDown bool
}

func NewAsyncProcessor(externalClient *external.Client, workerCount int) *AsyncProcessor {
	if workerCount <= 0 {
		workerCount = 3
	}

	ctx, cancel := context.WithCancel(context.Background())

	processor := &AsyncProcessor{
		externalClient: externalClient,
		jobQueue:       make(chan ExtractionJob, 100),
		workerCount:    workerCount,
		ctx:            ctx,
		cancel:         cancel,
		isShuttingDown: false,
	}

	for i := 0; i < workerCount; i++ {
		processor.wg.Add(1)
		go processor.worker(i)
	}

	return processor
}

func (p *AsyncProcessor) worker(id int) {
	defer p.wg.Done()
	log.Printf("Extraction worker %d started", id)

	for {
		select {
		case <-p.ctx.Done():
			log.Printf("Extraction worker %d received shutdown signal", id)
			return
		case job, ok := <-p.jobQueue:
			if !ok {
				log.Printf("Extraction worker %d: job queue closed", id)
				return
			}

			log.Printf("Worker %d: Processing extraction job for detail ID %d", id, job.DetailID)

			err := p.externalClient.ExtractDocument(job.Request)
			if err != nil {
				log.Printf("Worker %d: Failed to extract document (detail ID: %d): %v", id, job.DetailID, err)
			} else {
				log.Printf("Worker %d: Successfully extracted document (detail ID: %d)", id, job.DetailID)
			}
		}
	}
}

func (p *AsyncProcessor) SubmitJob(job ExtractionJob) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.isShuttingDown {
		return fmt.Errorf("processor is shutting down, cannot accept new jobs")
	}

	select {
	case p.jobQueue <- job:
		log.Printf("Extraction job submitted for detail ID %d (queue size: %d)", job.DetailID, len(p.jobQueue))
		return nil
	default:
		return fmt.Errorf("job queue is full (%d jobs), cannot submit new job", cap(p.jobQueue))
	}
}

func (p *AsyncProcessor) Shutdown() {
	p.mu.Lock()
	p.isShuttingDown = true
	p.mu.Unlock()

	log.Println("Shutting down async processor...")

	p.cancel()

	close(p.jobQueue)

	p.wg.Wait()

	log.Println("Async processor shut down complete")
}

func (p *AsyncProcessor) GetQueueSize() int {
	return len(p.jobQueue)
}
