package process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"
)

// TailLines reads the last n lines from a file.
func TailLines(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("logs: open %s: %w", path, err)
	}
	defer f.Close()

	// Read all lines (log files are typically small)
	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("logs: scan: %w", err)
	}

	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

// Follow streams new lines from a file by polling, sending each line to the out channel.
// Returns when ctx is done.
func Follow(path string, out chan<- string, stop <-chan struct{}) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("logs: open %s: %w", path, err)
	}
	defer f.Close()

	// Seek to end
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	reader := bufio.NewReader(f)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var partial string
	for {
		select {
		case <-stop:
			return nil
		case <-ticker.C:
			for {
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					partial += line
					if len(partial) > 0 && partial[len(partial)-1] == '\n' {
						out <- partial[:len(partial)-1]
						partial = ""
					}
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
			}
		}
	}
}
