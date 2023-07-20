package lfs

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/soft-serve/git"
	gitm "github.com/gogs/git-module"
)

// SearchPointerBlobs scans the whole repository for LFS pointer files
func SearchPointerBlobs(ctx context.Context, repo *git.Repository, pointerChan chan<- PointerBlob, errChan chan<- error) {
	basePath := repo.Path

	catFileCheckReader, catFileCheckWriter := io.Pipe()
	shasToBatchReader, shasToBatchWriter := io.Pipe()
	catFileBatchReader, catFileBatchWriter := io.Pipe()

	wg := sync.WaitGroup{}
	wg.Add(6)

	// Create the go-routines in reverse order.

	// 4. Take the output of cat-file --batch and check if each file in turn
	// to see if they're pointers to files in the LFS store
	go createPointerResultsFromCatFileBatch(ctx, catFileBatchReader, &wg, pointerChan)

	// 3. Take the shas of the blobs and batch read them
	go catFileBatch(ctx, shasToBatchReader, catFileBatchWriter, &wg, basePath)

	// 2. From the provided objects restrict to blobs <=1k
	go blobsLessThan1024FromCatFileBatchCheck(catFileCheckReader, shasToBatchWriter, &wg)

	// 1. Run batch-check on all objects in the repository
	revListReader, revListWriter := io.Pipe()
	shasToCheckReader, shasToCheckWriter := io.Pipe()
	go catFileBatchCheck(ctx, shasToCheckReader, catFileCheckWriter, &wg, basePath)
	go blobsFromRevListObjects(revListReader, shasToCheckWriter, &wg)
	go revListAllObjects(ctx, revListWriter, &wg, basePath, errChan)
	wg.Wait()

	close(pointerChan)
	close(errChan)
}

func createPointerResultsFromCatFileBatch(ctx context.Context, catFileBatchReader *io.PipeReader, wg *sync.WaitGroup, pointerChan chan<- PointerBlob) {
	defer wg.Done()
	defer catFileBatchReader.Close() // nolint: errcheck

	bufferedReader := bufio.NewReader(catFileBatchReader)
	buf := make([]byte, 1025)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
		}

		// File descriptor line: sha
		sha, err := bufferedReader.ReadString(' ')
		if err != nil {
			_ = catFileBatchReader.CloseWithError(err)
			break
		}
		sha = strings.TrimSpace(sha)
		// Throw away the blob
		if _, err := bufferedReader.ReadString(' '); err != nil {
			_ = catFileBatchReader.CloseWithError(err)
			break
		}
		sizeStr, err := bufferedReader.ReadString('\n')
		if err != nil {
			_ = catFileBatchReader.CloseWithError(err)
			break
		}
		size, err := strconv.Atoi(sizeStr[:len(sizeStr)-1])
		if err != nil {
			_ = catFileBatchReader.CloseWithError(err)
			break
		}
		pointerBuf := buf[:size+1]
		if _, err := io.ReadFull(bufferedReader, pointerBuf); err != nil {
			_ = catFileBatchReader.CloseWithError(err)
			break
		}
		pointerBuf = pointerBuf[:size]
		// Now we need to check if the pointerBuf is an LFS pointer
		pointer, _ := ReadPointerFromBuffer(pointerBuf)
		if !pointer.IsValid() {
			continue
		}

		pointerChan <- PointerBlob{Hash: sha, Pointer: pointer}
	}
}

func catFileBatch(ctx context.Context, shasToBatchReader *io.PipeReader, catFileBatchWriter *io.PipeWriter, wg *sync.WaitGroup, basePath string) {
	defer wg.Done()
	defer shasToBatchReader.Close()  // nolint: errcheck
	defer catFileBatchWriter.Close() // nolint: errcheck

	stderr := new(bytes.Buffer)
	var errbuf strings.Builder
	if err := gitm.NewCommandWithContext(ctx, "cat-file", "--batch").RunInDirWithOptions(basePath, gitm.RunInDirOptions{
		Stdout: catFileBatchWriter,
		Stdin:  shasToBatchReader,
		Stderr: stderr,
	}); err != nil {
		_ = shasToBatchReader.CloseWithError(fmt.Errorf("git rev-list [%s]: %w - %s", basePath, err, errbuf.String()))
	}
}

func blobsLessThan1024FromCatFileBatchCheck(catFileCheckReader *io.PipeReader, shasToBatchWriter *io.PipeWriter, wg *sync.WaitGroup) {
	defer wg.Done()
	defer catFileCheckReader.Close() // nolint: errcheck
	scanner := bufio.NewScanner(catFileCheckReader)
	defer func() {
		_ = shasToBatchWriter.CloseWithError(scanner.Err())
	}()
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		fields := strings.Split(line, " ")
		if len(fields) < 3 || fields[1] != "blob" {
			continue
		}
		size, _ := strconv.Atoi(fields[2])
		if size > 1024 {
			continue
		}
		toWrite := []byte(fields[0] + "\n")
		for len(toWrite) > 0 {
			n, err := shasToBatchWriter.Write(toWrite)
			if err != nil {
				_ = catFileCheckReader.CloseWithError(err)
				break
			}
			toWrite = toWrite[n:]
		}
	}
}

func catFileBatchCheck(ctx context.Context, shasToCheckReader *io.PipeReader, catFileCheckWriter *io.PipeWriter, wg *sync.WaitGroup, basePath string) {
	defer wg.Done()
	defer shasToCheckReader.Close()  // nolint: errcheck
	defer catFileCheckWriter.Close() // nolint: errcheck

	stderr := new(bytes.Buffer)
	var errbuf strings.Builder
	if err := gitm.NewCommandWithContext(ctx, "cat-file", "--batch-check").RunInDirWithOptions(basePath, gitm.RunInDirOptions{
		Stdout: catFileCheckWriter,
		Stdin:  shasToCheckReader,
		Stderr: stderr,
	}); err != nil {
		_ = shasToCheckReader.CloseWithError(fmt.Errorf("git rev-list [%s]: %w - %s", basePath, err, errbuf.String()))
	}
}

func blobsFromRevListObjects(revListReader *io.PipeReader, shasToCheckWriter *io.PipeWriter, wg *sync.WaitGroup) {
	defer wg.Done()
	defer revListReader.Close() // nolint: errcheck
	scanner := bufio.NewScanner(revListReader)
	defer func() {
		_ = shasToCheckWriter.CloseWithError(scanner.Err())
	}()

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		fields := strings.Split(line, " ")
		if len(fields) < 2 || len(fields[1]) == 0 {
			continue
		}
		toWrite := []byte(fields[0] + "\n")
		for len(toWrite) > 0 {
			n, err := shasToCheckWriter.Write(toWrite)
			if err != nil {
				_ = revListReader.CloseWithError(err)
				break
			}
			toWrite = toWrite[n:]
		}
	}
}

func revListAllObjects(ctx context.Context, revListWriter *io.PipeWriter, wg *sync.WaitGroup, basePath string, errChan chan<- error) {
	defer wg.Done()
	defer revListWriter.Close() // nolint: errcheck

	stderr := new(bytes.Buffer)
	var errbuf strings.Builder
	if err := gitm.NewCommandWithContext(ctx, "rev-list", "--objects", "--all").RunInDirWithOptions(basePath, gitm.RunInDirOptions{
		Stdout: revListWriter,
		Stderr: stderr,
	}); err != nil {
		errChan <- fmt.Errorf("git rev-list [%s]: %w - %s", basePath, err, errbuf.String())
	}
}
