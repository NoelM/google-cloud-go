// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"

	gax "github.com/googleapis/gax-go/v2"
	"google.golang.org/api/option"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

// TODO(noahdietz): Move existing factory methods to this file.

// storageClient is an internal-only interface designed to separate the
// transport-specific logic of making Storage API calls from the logic of the
// client library.
//
// Implementation requirements beyond implementing the interface include:
// * factory method(s) must accept a `userProject string` param
// * `settings` must be retained per instance
// * `storageOption`s must be resolved in the order they are received
// * all API errors must be wrapped in the gax-go APIError type
// * any unimplemented interface methods must return a StorageUnimplementedErr
//
// TODO(noahdietz): This interface is currently not used in the production code
// paths
type storageClient interface {

	// Top-level methods.

	GetServiceAccount(ctx context.Context, project string, opts ...storageOption) (string, error)
	CreateBucket(ctx context.Context, project string, attrs *BucketAttrs, opts ...storageOption) (*BucketAttrs, error)
	ListBuckets(ctx context.Context, project string, opts ...storageOption) (*BucketIterator, error)

	// Bucket methods.

	DeleteBucket(ctx context.Context, bucket string, conds *BucketConditions, opts ...storageOption) error
	GetBucket(ctx context.Context, bucket string, conds *BucketConditions, opts ...storageOption) (*BucketAttrs, error)
	UpdateBucket(ctx context.Context, uattrs *BucketAttrsToUpdate, conds *BucketConditions, opts ...storageOption) (*BucketAttrs, error)
	LockBucketRetentionPolicy(ctx context.Context, bucket string, conds *BucketConditions, opts ...storageOption) error
	ListObjects(ctx context.Context, bucket string, q *Query, opts ...storageOption) (*ObjectIterator, error)

	// Object metadata methods.

	DeleteObject(ctx context.Context, bucket, object string, conds *Conditions, opts ...storageOption) error
	GetObject(ctx context.Context, bucket, object string, conds *Conditions, opts ...storageOption) (*ObjectAttrs, error)
	UpdateObject(ctx context.Context, bucket, object string, uattrs *ObjectAttrsToUpdate, conds *Conditions, opts ...storageOption) (*ObjectAttrs, error)

	// Default Object ACL methods.

	DeleteDefaultObjectACL(ctx context.Context, bucket string, entity ACLEntity, opts ...storageOption) error
	ListDefaultObjectACLs(ctx context.Context, bucket string, opts ...storageOption) ([]ACLRule, error)
	UpdateDefaultObjectACL(ctx context.Context, opts ...storageOption) (*ACLRule, error)

	// Bucket ACL methods.

	DeleteBucketACL(ctx context.Context, bucket string, entity ACLEntity, opts ...storageOption) error
	ListBucketACLs(ctx context.Context, bucket string, opts ...storageOption) ([]ACLRule, error)
	UpdateBucketACL(ctx context.Context, bucket string, entity ACLEntity, role ACLRole, opts ...storageOption) (*ACLRule, error)

	// Object ACL methods.

	DeleteObjectACL(ctx context.Context, bucket, object string, entity ACLEntity, opts ...storageOption) error
	ListObjectACLs(ctx context.Context, bucket, object string, opts ...storageOption) ([]ACLRule, error)
	UpdateObjectACL(ctx context.Context, bucket, object string, entity ACLEntity, role ACLRole, opts ...storageOption) (*ACLRule, error)

	// Media operations.

	ComposeObject(ctx context.Context, req *composeObjectRequest, opts ...storageOption) (*ObjectAttrs, error)
	RewriteObject(ctx context.Context, req *rewriteObjectRequest, opts ...storageOption) (*rewriteObjectResponse, error)

	OpenReader(ctx context.Context, r *Reader, opts ...storageOption) error
	OpenWriter(ctx context.Context, w *Writer, opts ...storageOption) error

	// IAM methods.

	GetIamPolicy(ctx context.Context, resource string, version int32, opts ...storageOption) (*iampb.Policy, error)
	SetIamPolicy(ctx context.Context, resource string, policy *iampb.Policy, opts ...storageOption) error
	TestIamPermissions(ctx context.Context, resource string, permissions []string, opts ...storageOption) ([]string, error)

	// HMAC Key methods.

	GetHMACKey(ctx context.Context, desc *hmacKeyDesc, opts ...storageOption) (*HMACKey, error)
	ListHMACKey(ctx context.Context, desc *hmacKeyDesc, opts ...storageOption) *HMACKeysIterator
	UpdateHMACKey(ctx context.Context, desc *hmacKeyDesc, attrs *HMACKeyAttrsToUpdate, opts ...storageOption) (*HMACKey, error)
	CreateHMACKey(ctx context.Context, desc *hmacKeyDesc, opts ...storageOption) (*HMACKey, error)
	DeleteHMACKey(ctx context.Context, desc *hmacKeyDesc, opts ...storageOption) error
}

// settings contains transport-agnostic configuration for API calls made via
// the storageClient inteface. All implementations must utilize settings
// and respect those that are applicable.
type settings struct {
	// retry is the complete retry configuration to use when evaluating if an
	// API call should be retried.
	retry *retryConfig

	// gax is a set of gax.CallOption to be conveyed to gax.Invoke.
	// Note: Not all storageClient interfaces will must use gax.Invoke.
	gax []gax.CallOption

	// idempotent indicates if the call is idempotent or not when considering
	// if the call should be retired or not.
	idempotent bool

	// clientOption is a set of option.ClientOption to be used during client
	// transport initialization. See https://pkg.go.dev/google.golang.org/api/option
	// for a list of supported options.
	clientOption []option.ClientOption
}

func initSettings(opts ...storageOption) *settings {
	s := &settings{}
	resolveOptions(s, opts...)
	return s
}

func resolveOptions(s *settings, opts ...storageOption) {
	for _, o := range opts {
		o.Apply(s)
	}
}

// callSettings is a helper for resolving storage options against the settings
// in the context of an individual call. This is to ensure that client-level
// default settings are not mutated by two different calls getting options.
//
// Example: s := callSettings(c.settings, opts...)
func callSettings(defaults *settings, opts ...storageOption) *settings {
	if defaults == nil {
		return nil
	}
	// This does not make a deep copy of the pointer/slice fields, but all
	// options replace the settings fields rather than modify their values in
	// place.
	cs := *defaults
	resolveOptions(&cs, opts...)
	return &cs
}

// storageOption is the transport-agnostic call option for the storageClient
// interface.
type storageOption interface {
	Apply(s *settings)
}

func withGAXOptions(opts ...gax.CallOption) storageOption {
	return &gaxOption{opts}
}

type gaxOption struct {
	opts []gax.CallOption
}

func (o *gaxOption) Apply(s *settings) { s.gax = o.opts }

func withRetryConfig(rc *retryConfig) storageOption {
	return &retryOption{rc}
}

type retryOption struct {
	rc *retryConfig
}

func (o *retryOption) Apply(s *settings) { s.retry = o.rc }

func idempotent(i bool) storageOption {
	return &idempotentOption{i}
}

type idempotentOption struct {
	idempotency bool
}

func (o *idempotentOption) Apply(s *settings) { s.idempotent = o.idempotency }

func withClientOptions(opts ...option.ClientOption) storageOption {
	return &clientOption{opts: opts}
}

type clientOption struct {
	opts []option.ClientOption
}

func (o *clientOption) Apply(s *settings) { s.clientOption = o.opts }

type composeObjectRequest struct {
	dstBucket     string
	dstObject     string
	srcs          []string
	gen           int64
	conds         *Conditions
	predefinedACL string
}

type rewriteObjectRequest struct {
	srcBucket     string
	srcObject     string
	dstBucket     string
	dstObject     string
	dstKeyName    string
	attrs         *ObjectAttrs
	gen           int64
	conds         *Conditions
	predefinedACL string
	token         string
}

type rewriteObjectResponse struct {
	resource *ObjectAttrs
	done     bool
	written  int64
	token    string
}
