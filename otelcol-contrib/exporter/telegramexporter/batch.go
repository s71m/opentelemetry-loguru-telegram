// batch.go
package telegramexporter

import (
    "sync"
    "time"
)

type messageItem struct {
    content   string
    threadID  int
    timestamp time.Time
}

type batch struct {
    messages []*messageItem
    created  time.Time
}

type batchProcessor struct {
    mu       sync.Mutex
    batches  map[int]*batch
    config   *Config
    sendFunc func(messages []*messageItem) error
    done     chan struct{}
}

func newBatchProcessor(cfg *Config, sendFunc func(messages []*messageItem) error) *batchProcessor {
    bp := &batchProcessor{
        batches:  make(map[int]*batch),
        config:   cfg,
        sendFunc: sendFunc,
        done:     make(chan struct{}),
    }

    if cfg.BatchEnabled {
        go bp.startBatchTimer()
    }

    return bp
}

func (bp *batchProcessor) startBatchTimer() {
    ticker := time.NewTicker(bp.config.BatchTimeout)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            bp.flush()
        case <-bp.done:
            return
        }
    }
}

func (bp *batchProcessor) add(msg string, threadID int) error {
    if !bp.config.BatchEnabled {
        return bp.sendFunc([]*messageItem{{content: msg, threadID: threadID, timestamp: time.Now()}})
    }

    bp.mu.Lock()
    defer bp.mu.Unlock()

    if _, exists := bp.batches[threadID]; !exists {
        bp.batches[threadID] = &batch{
            messages: make([]*messageItem, 0, bp.config.BatchSize),
            created:  time.Now(),
        }
    }

    bp.batches[threadID].messages = append(bp.batches[threadID].messages, &messageItem{
        content:   msg,
        threadID:  threadID,
        timestamp: time.Now(),
    })

    if len(bp.batches[threadID].messages) >= bp.config.BatchSize {
        return bp.flushBatch(threadID)
    }

    return nil
}

func (bp *batchProcessor) flush() {
    bp.mu.Lock()
    defer bp.mu.Unlock()

    for threadID := range bp.batches {
        _ = bp.flushBatch(threadID)
    }
}

func (bp *batchProcessor) flushBatch(threadID int) error {
    if batch, exists := bp.batches[threadID]; exists && len(batch.messages) > 0 {
        err := bp.sendFunc(batch.messages)
        if err == nil {
            delete(bp.batches, threadID)
        }
        return err
    }
    return nil
}

func (bp *batchProcessor) stop() {
    close(bp.done)
    bp.flush()
}