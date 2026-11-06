package watch

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/execute/watchmanager"
	"github.com/microsoft/TypeScript/tsc/internal/fswatch"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
)

// pollingBackend is what --usePolling asks of chokidar. fswatch only has native notifications, so this walks the
// watched folders on a timer. It's for filesystems where those don't work, such as network shares.
type pollingBackend struct {
	interval time.Duration
}

func newPollingBackend(interval time.Duration) *pollingBackend {
	return &pollingBackend{interval: interval}
}

type polledEntry struct {
	modTime time.Time
	size    int64
	isDir   bool
}

type pollWatch struct {
	stop chan struct{}
	done chan struct{}
	once sync.Once
}

func (w *pollWatch) Close() error {
	w.once.Do(func() { close(w.stop) })
	<-w.done
	return nil
}

func (b *pollingBackend) WatchDirectories(requests []watchmanager.WatchDirectoryRequest) ([]io.Closer, error) {
	closers := make([]io.Closer, len(requests))
	for i, request := range requests {
		// The first snapshot is taken here so that a change made right after this returns gets seen.
		previous := snapshotDirectory(request)
		watch := &pollWatch{stop: make(chan struct{}), done: make(chan struct{})}
		closers[i] = watch
		go func() {
			defer close(watch.done)
			ticker := time.NewTicker(b.interval)
			defer ticker.Stop()
			for {
				select {
				case <-watch.stop:
					return
				case <-ticker.C:
				}
				current := snapshotDirectory(request)
				if events := diffSnapshots(previous, current); len(events) > 0 {
					request.Callback(events, nil)
				}
				previous = current
			}
		}()
	}
	return closers, nil
}

func snapshotDirectory(request watchmanager.WatchDirectoryRequest) map[string]polledEntry {
	result := map[string]polledEntry{}
	root := filepath.FromSlash(request.Dir.AsString())
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == root {
			return nil
		}
		slashed := filepath.ToSlash(path)
		if request.Ignore != nil && request.Ignore(tspath.RootedPathFromAbsolute(slashed)) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return nil
		}
		// A folder's mtime changes with its contents, which get their own events.
		if info.IsDir() {
			result[slashed] = polledEntry{isDir: true}
		} else {
			result[slashed] = polledEntry{modTime: info.ModTime(), size: info.Size()}
		}
		if entry.IsDir() && !request.Recursive {
			return fs.SkipDir
		}
		return nil
	})
	return result
}

func diffSnapshots(previous, current map[string]polledEntry) []watchmanager.WatchEvent {
	var events []watchmanager.WatchEvent
	for path, entry := range current {
		if old, ok := previous[path]; !ok || old != entry {
			events = append(events, watchmanager.WatchEvent{Path: tspath.RootedPathFromAbsolute(path), Kind: fswatch.EventUpdate})
		}
	}
	for path := range previous {
		if _, ok := current[path]; !ok {
			events = append(events, watchmanager.WatchEvent{Path: tspath.RootedPathFromAbsolute(path), Kind: fswatch.EventDelete})
		}
	}
	return events
}
