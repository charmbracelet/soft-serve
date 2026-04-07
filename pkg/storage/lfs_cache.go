package storage

import (
	"context"
	"path/filepath"
	"sync"
)

var lfsStorageCache sync.Map // map[string]Storage

func lfsStorageCacheKey(dataPath, repoID string) string {
	return dataPath + "/" + repoID
}

// GetLFSStorage returns a cached Storage for the given repo.
// If s3cfg is non-nil, returns an S3Storage; otherwise returns a LocalStorage.
func GetLFSStorage(dataPath string, repoID string, s3cfg *S3Config) Storage {
	if s3cfg != nil {
		key := "s3:" + s3cfg.Endpoint + "/" + s3cfg.Bucket + "/" + repoID
		if v, ok := lfsStorageCache.Load(key); ok {
			return v.(Storage)
		}
		prefix := s3cfg.Prefix
		if prefix != "" && prefix[len(prefix)-1] != '/' {
			prefix += "/"
		}
		prefix += repoID + "/"
		cfg := S3Config{
			Endpoint:  s3cfg.Endpoint,
			Region:    s3cfg.Region,
			Bucket:    s3cfg.Bucket,
			Prefix:    prefix,
			AccessKey: s3cfg.AccessKey,
			SecretKey: s3cfg.SecretKey,
		}
		s3, err := NewS3Storage(context.Background(), cfg)
		if err != nil {
			return NewLocalStorage(filepath.Join(dataPath, "lfs", repoID))
		}
		lfsStorageCache.Store(key, s3)
		return s3
	}

	key := lfsStorageCacheKey(dataPath, repoID)
	if v, ok := lfsStorageCache.Load(key); ok {
		return v.(Storage)
	}
	strg := NewLocalStorage(filepath.Join(dataPath, "lfs", repoID))
	lfsStorageCache.Store(key, strg)
	return strg
}

// DeleteLFSStorage removes the cached Storage for a repo.
func DeleteLFSStorage(dataPath, repoID string, s3cfg *S3Config) {
	if s3cfg != nil {
		key := "s3:" + s3cfg.Endpoint + "/" + s3cfg.Bucket + "/" + repoID
		lfsStorageCache.Delete(key)
		return
	}
	lfsStorageCache.Delete(lfsStorageCacheKey(dataPath, repoID))
}
