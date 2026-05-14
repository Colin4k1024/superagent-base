/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package impl

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/superagent-ai/superagent-base/backend/infra/imagex"
	"github.com/superagent-ai/superagent-base/backend/infra/storage"
	"github.com/superagent-ai/superagent-base/backend/infra/storage/impl/minio"
	"github.com/superagent-ai/superagent-base/backend/infra/storage/impl/s3"
	"github.com/superagent-ai/superagent-base/backend/infra/storage/impl/tos"
	"github.com/superagent-ai/superagent-base/backend/types/consts"
)

type Storage = storage.Storage

func New(ctx context.Context) (Storage, error) {
	storageType := os.Getenv(consts.StorageType)
	switch storageType {
	case "minio":
		return minio.New(
			ctx,
			os.Getenv(consts.MinIOEndpoint),
			os.Getenv(consts.MinIOAK),
			os.Getenv(consts.MinIOSK),
			os.Getenv(consts.StorageBucket),
			false,
		)
	case "tos":
		return tos.New(
			ctx,
			os.Getenv(consts.TOSAccessKey),
			os.Getenv(consts.TOSSecretKey),
			os.Getenv(consts.StorageBucket),
			os.Getenv(consts.TOSEndpoint),
			os.Getenv(consts.TOSRegion),
		)
	case "s3":
		return s3.New(
			ctx,
			os.Getenv(consts.S3AccessKey),
			os.Getenv(consts.S3SecretKey),
			os.Getenv(consts.StorageBucket),
			os.Getenv(consts.S3Endpoint),
			os.Getenv(consts.S3Region),
		)
	case "none", "":
		return &noopStorage{}, nil
	}

	return nil, fmt.Errorf("unknown storage type: %s", storageType)
}

// noopStorage is a no-op storage for environments without object storage (dev/test).
// All write operations return an error; read operations return empty results.
type noopStorage struct{}

var errStorageDisabled = fmt.Errorf("storage disabled (STORAGE_TYPE=none)")

func (n *noopStorage) PutObject(_ context.Context, _ string, _ []byte, _ ...storage.PutOptFn) error {
	return errStorageDisabled
}
func (n *noopStorage) PutObjectWithReader(_ context.Context, _ string, _ io.Reader, _ ...storage.PutOptFn) error {
	return errStorageDisabled
}
func (n *noopStorage) GetObject(_ context.Context, _ string) ([]byte, error) {
	return nil, errStorageDisabled
}
func (n *noopStorage) DeleteObject(_ context.Context, _ string) error {
	return errStorageDisabled
}
func (n *noopStorage) GetObjectUrl(_ context.Context, key string, _ ...storage.GetOptFn) (string, error) {
	return key, nil
}
func (n *noopStorage) HeadObject(_ context.Context, _ string, _ ...storage.GetOptFn) (*storage.FileInfo, error) {
	return nil, errStorageDisabled
}
func (n *noopStorage) ListAllObjects(_ context.Context, _ string, _ ...storage.GetOptFn) ([]*storage.FileInfo, error) {
	return nil, errStorageDisabled
}
func (n *noopStorage) ListObjectsPaginated(_ context.Context, _ *storage.ListObjectsPaginatedInput, _ ...storage.GetOptFn) (*storage.ListObjectsPaginatedOutput, error) {
	return nil, errStorageDisabled
}

// NewNoop returns a no-op Storage that silently accepts writes and returns empty results.
// Use this as a fallback when real storage (MinIO/TOS/S3) is unavailable in dev/test environments.
func NewNoop() Storage {
	return &noopStorage{}
}

func NewImagex(ctx context.Context) (imagex.ImageX, error) {
	storageType := os.Getenv(consts.StorageType)
	switch storageType {
	case "minio":
		return minio.NewStorageImagex(
			ctx,
			os.Getenv(consts.MinIOEndpoint),
			os.Getenv(consts.MinIOAK),
			os.Getenv(consts.MinIOSK),
			os.Getenv(consts.StorageBucket),
			false,
		)
	case "tos":
		return tos.NewStorageImagex(
			ctx,
			os.Getenv(consts.TOSAccessKey),
			os.Getenv(consts.TOSSecretKey),
			os.Getenv(consts.StorageBucket),
			os.Getenv(consts.TOSEndpoint),
			os.Getenv(consts.TOSRegion),
		)
	case "s3":
		return s3.NewStorageImagex(
			ctx,
			os.Getenv(consts.S3AccessKey),
			os.Getenv(consts.S3SecretKey),
			os.Getenv(consts.StorageBucket),
			os.Getenv(consts.S3Endpoint),
			os.Getenv(consts.S3Region),
		)
	case "none", "":
		return nil, nil
	}
	return nil, fmt.Errorf("unknown storage type: %s", storageType)
}
