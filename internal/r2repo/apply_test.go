package r2repo

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyUsesOrderedPlanAndVerifiesRemoteContent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeCandidate(t, root, "apt/pool/fixture/new.deb", "new package")
	writeCandidate(t, root, "apt/dists/stable/fixture/binary-amd64/Packages.gz", "new index")
	writeCandidate(t, root, "apt/dists/stable/InRelease", "new activation")
	writeCandidate(t, root, "_state/manifest.json", "new state")
	client := &fakeS3Client{objects: map[string][]byte{
		"_staging/apt/pool/fixture/old.deb":                          []byte("old package"),
		"_staging/apt/dists/stable/fixture/binary-amd64/Packages.gz": []byte("old index"),
		"_staging/apt/dists/stable/InRelease":                        []byte("old activation"),
		"_staging/_state/manifest.json":                              []byte("old state"),
	}}
	request := validRequest(root)

	result, err := applyWithClient(context.Background(), client, request)

	require.NoError(t, err)
	assert.True(t, result.Verified)
	assert.False(t, result.NoOp)
	assert.Equal(t, []string{
		"put:_staging/apt/pool/fixture/new.deb",
		"put:_staging/apt/dists/stable/fixture/binary-amd64/Packages.gz",
		"put:_staging/apt/dists/stable/InRelease",
		"put:_staging/_state/manifest.json",
		"delete:_staging/apt/pool/fixture/old.deb",
	}, client.mutations)
	assert.Equal(t, "new package", string(client.objects["_staging/apt/pool/fixture/new.deb"]))
	assert.NotContains(t, client.objects, "_staging/apt/pool/fixture/old.deb")
	assert.Equal(t, "no-store", client.cacheControl)
}

func TestApplyReturnsVerifiedNoOp(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeCandidate(t, root, "meigma.asc", "public key")
	client := &fakeS3Client{objects: map[string][]byte{"_staging/meigma.asc": []byte("public key")}}

	result, err := applyWithClient(context.Background(), client, validRequest(root))

	require.NoError(t, err)
	assert.True(t, result.NoOp)
	assert.True(t, result.Verified)
	assert.Empty(t, result.Actions)
	assert.Empty(t, client.mutations)
}

func TestRequestValidationRejectsUnsafeConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Request)
		message string
	}{
		{
			name:    "missing secret key",
			mutate:  func(request *Request) { request.SecretAccessKey = "" },
			message: "R2 secret key is required",
		},
		{
			name:    "absolute prefix",
			mutate:  func(request *Request) { request.Prefix = "/_staging/" },
			message: "prefix must be a clean relative path",
		},
		{
			name:    "traversing prefix",
			mutate:  func(request *Request) { request.Prefix = "../_staging/" },
			message: "prefix must be a clean relative path",
		},
		{
			name:    "prefix without slash",
			mutate:  func(request *Request) { request.Prefix = "_staging" },
			message: "prefix must be a clean relative path",
		},
		{
			name:    "insecure endpoint",
			mutate:  func(request *Request) { request.Endpoint = "http://r2.example" },
			message: "endpoint must use https",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := validRequest(t.TempDir())
			test.mutate(&request)

			err := request.validate()

			require.Error(t, err)
			assert.Contains(t, err.Error(), test.message)
		})
	}
}

func writeCandidate(t *testing.T, root string, relative string, content string) {
	t.Helper()

	file := filepath.Join(root, filepath.FromSlash(relative))
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
	require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
}

func validRequest(root string) Request {
	return Request{
		Root:            root,
		Bucket:          "meigma-packages",
		Prefix:          "_staging/",
		Endpoint:        "https://example.r2.cloudflarestorage.com",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		SessionToken:    "session-token",
	}
}

type fakeS3Client struct {
	objects      map[string][]byte
	mutations    []string
	cacheControl string
}

func (client *fakeS3Client) ListObjectsV2(
	_ context.Context,
	input *s3.ListObjectsV2Input,
	_ ...func(*s3.Options),
) (*s3.ListObjectsV2Output, error) {
	keys := make([]string, 0, len(client.objects))
	for key := range client.objects {
		if strings.HasPrefix(key, aws.ToString(input.Prefix)) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	contents := make([]types.Object, 0, len(keys))
	for _, key := range keys {
		contents = append(contents, types.Object{Key: aws.String(key)})
	}

	return &s3.ListObjectsV2Output{Contents: contents}, nil
}

func (client *fakeS3Client) GetObject(
	_ context.Context,
	input *s3.GetObjectInput,
	_ ...func(*s3.Options),
) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(client.objects[aws.ToString(input.Key)]))}, nil
}

func (client *fakeS3Client) PutObject(
	_ context.Context,
	input *s3.PutObjectInput,
	_ ...func(*s3.Options),
) (*s3.PutObjectOutput, error) {
	content, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	key := aws.ToString(input.Key)
	client.objects[key] = content
	client.mutations = append(client.mutations, "put:"+key)
	client.cacheControl = aws.ToString(input.CacheControl)

	return &s3.PutObjectOutput{}, nil
}

func (client *fakeS3Client) DeleteObject(
	_ context.Context,
	input *s3.DeleteObjectInput,
	_ ...func(*s3.Options),
) (*s3.DeleteObjectOutput, error) {
	key := aws.ToString(input.Key)
	delete(client.objects, key)
	client.mutations = append(client.mutations, "delete:"+key)

	return &s3.DeleteObjectOutput{}, nil
}
