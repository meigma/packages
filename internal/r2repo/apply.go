// Package r2repo applies verified package repository trees to Cloudflare R2.
package r2repo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/meigma/packages/internal/localrepo"
)

// Request describes one ordered candidate-tree application to R2.
type Request struct {
	// Root is the verified local candidate tree.
	Root string
	// Bucket is the exact R2 bucket to mutate.
	Bucket string
	// Prefix confines every remote object operation.
	Prefix string
	// ProductionRoot enables explicit bucket-root publication while preserving reserved staging objects.
	ProductionRoot bool
	// Endpoint is the account-specific R2 S3 endpoint.
	Endpoint string
	// AccessKeyID authenticates the S3 client.
	AccessKeyID string
	// SecretAccessKey authenticates the S3 client.
	SecretAccessKey string
	// SessionToken carries optional temporary-credential state.
	SessionToken string
}

// Result summarizes an ordered and remotely verified R2 application.
type Result struct {
	// Bucket is the R2 bucket that was checked or mutated.
	Bucket string `json:"bucket"`
	// Prefix is the confined object prefix within the bucket.
	Prefix string `json:"prefix"`
	// Actions lists the ordered creates, replacements, and deletions applied.
	Actions []localrepo.SyncAction `json:"actions"`
	// NoOp reports whether the verified candidate already matched R2.
	NoOp bool `json:"no_op"`
	// Verified reports whether the final remote snapshot matched the candidate.
	Verified bool `json:"verified"`
}

type s3Client interface {
	ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// Apply hydrates the current prefix, applies the existing ordered sync plan,
// and hydrates it again to verify the final remote content.
func Apply(ctx context.Context, request Request) (Result, error) {
	if err := request.validate(); err != nil {
		return Result{}, err
	}
	awsConfig, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			request.AccessKeyID,
			request.SecretAccessKey,
			request.SessionToken,
		)),
	)
	if err != nil {
		return Result{}, fmt.Errorf("load R2 client configuration: %w", err)
	}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(request.Endpoint)
		options.UsePathStyle = true
	})

	return applyWithClient(ctx, client, request)
}

func applyWithClient(ctx context.Context, client s3Client, request Request) (Result, error) {
	if request.ProductionRoot {
		if err := validateProductionCandidate(request.Root); err != nil {
			return Result{}, err
		}
	}
	remoteRoot, err := os.MkdirTemp("", "meigma-packages-r2-before-")
	if err != nil {
		return Result{}, fmt.Errorf("create remote snapshot: %w", err)
	}
	defer os.RemoveAll(remoteRoot)
	if hydrateErr := hydrate(ctx, client, request, remoteRoot); hydrateErr != nil {
		return Result{}, hydrateErr
	}
	plan, err := localrepo.PlanSync(request.Root, remoteRoot)
	if err != nil {
		return Result{}, fmt.Errorf("plan R2 sync: %w", err)
	}
	if applyErr := applyPlan(ctx, client, request, plan); applyErr != nil {
		return Result{}, applyErr
	}

	verifiedRoot, err := os.MkdirTemp("", "meigma-packages-r2-after-")
	if err != nil {
		return Result{}, fmt.Errorf("create verification snapshot: %w", err)
	}
	defer os.RemoveAll(verifiedRoot)
	if hydrateErr := hydrate(ctx, client, request, verifiedRoot); hydrateErr != nil {
		return Result{}, fmt.Errorf("verify R2 prefix: %w", hydrateErr)
	}
	verification, err := localrepo.PlanSync(request.Root, verifiedRoot)
	if err != nil {
		return Result{}, fmt.Errorf("compare verified R2 prefix: %w", err)
	}
	if len(verification.Actions) != 0 {
		return Result{}, fmt.Errorf("verify R2 prefix: remote content differs by %d actions", len(verification.Actions))
	}

	return Result{
		Bucket:   request.Bucket,
		Prefix:   request.Prefix,
		Actions:  plan.Actions,
		NoOp:     len(plan.Actions) == 0,
		Verified: true,
	}, nil
}

func (request Request) validate() error {
	fields := map[string]string{
		"root":             request.Root,
		"bucket":           request.Bucket,
		"endpoint":         request.Endpoint,
		"R2 access key ID": request.AccessKeyID,
		"R2 secret key":    request.SecretAccessKey,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if request.ProductionRoot {
		if request.Prefix != "" {
			return errors.New("production-root publication requires an empty prefix")
		}
	} else if strings.TrimSpace(request.Prefix) == "" {
		return errors.New("prefix is required unless production-root publication is enabled")
	}
	if !request.ProductionRoot {
		if strings.HasPrefix(request.Prefix, "/") || !strings.HasSuffix(request.Prefix, "/") ||
			strings.HasPrefix(request.Prefix, "../") || strings.Contains(request.Prefix, "/../") ||
			path.Clean(request.Prefix)+"/" != request.Prefix {
			return errors.New("prefix must be a clean relative path ending in a slash")
		}
	}
	if !strings.HasPrefix(request.Endpoint, "https://") {
		return errors.New("endpoint must use https")
	}

	return nil
}

func validateProductionCandidate(root string) error {
	required := []string{
		"meigma.asc",
		filepath.Join("_state", "manifest.json"),
		filepath.Join("apt", "dists", "stable", "InRelease"),
	}
	for _, relative := range required {
		info, err := os.Lstat(filepath.Join(root, relative))
		if err != nil {
			return fmt.Errorf("production candidate requires %s: %w", filepath.ToSlash(relative), err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("production candidate requires regular file %s", filepath.ToSlash(relative))
		}
	}
	if _, err := os.Lstat(filepath.Join(root, "_staging")); err == nil {
		return errors.New("production candidate must not contain the reserved _staging path")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect reserved production candidate path: %w", err)
	}
	repomd, err := filepath.Glob(filepath.Join(root, "rpm", "*", "repodata", "repomd.xml"))
	if err != nil {
		return fmt.Errorf("inspect production RPM metadata: %w", err)
	}
	if len(repomd) == 0 {
		return errors.New("production candidate requires at least one RPM repomd.xml")
	}

	return nil
}

func hydrate(ctx context.Context, client s3Client, request Request, root string) error {
	var continuationToken *string
	for {
		output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(request.Bucket),
			Prefix:            aws.String(request.Prefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return fmt.Errorf("list R2 prefix: %w", err)
		}
		if err := hydrateObjects(ctx, client, request, root, objectKeys(output)); err != nil {
			return err
		}
		if !aws.ToBool(output.IsTruncated) {
			return nil
		}
		if output.NextContinuationToken == nil {
			return errors.New("list R2 prefix: truncated response omitted continuation token")
		}
		continuationToken = output.NextContinuationToken
	}
}

func hydrateObjects(
	ctx context.Context,
	client s3Client,
	request Request,
	root string,
	objects []typesObject,
) error {
	objects = append([]typesObject(nil), objects...)
	sort.Slice(objects, func(left, right int) bool { return objects[left].key < objects[right].key })
	for _, object := range objects {
		if !request.managesKey(object.key) {
			continue
		}
		relative, err := relativeObjectPath(request.Prefix, object.key)
		if err != nil {
			return err
		}
		if relative == "" {
			continue
		}
		if err := downloadObject(ctx, client, request.Bucket, object.key, root, relative); err != nil {
			return err
		}
	}

	return nil
}

func (request Request) managesKey(key string) bool {
	if !request.ProductionRoot {
		return strings.HasPrefix(key, request.Prefix)
	}

	return key != "_staging" && !strings.HasPrefix(key, "_staging/")
}

type typesObject struct {
	key string
}

func objectKeys(output *s3.ListObjectsV2Output) []typesObject {
	objects := make([]typesObject, 0, len(output.Contents))
	for _, object := range output.Contents {
		objects = append(objects, typesObject{key: aws.ToString(object.Key)})
	}

	return objects
}

func relativeObjectPath(prefix string, key string) (string, error) {
	if !strings.HasPrefix(key, prefix) {
		return "", fmt.Errorf("R2 object %q is outside prefix %q", key, prefix)
	}
	relative := strings.TrimPrefix(key, prefix)
	if relative == "" {
		return "", nil
	}
	if strings.HasPrefix(relative, "/") || path.Clean(relative) != relative || relative == "." {
		return "", fmt.Errorf("R2 object %q has unsafe relative path", key)
	}

	return relative, nil
}

func downloadObject(
	ctx context.Context,
	client s3Client,
	bucket string,
	key string,
	root string,
	relative string,
) error {
	output, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return fmt.Errorf("download R2 object %s: %w", key, err)
	}
	defer output.Body.Close()
	destination := filepath.Join(root, filepath.FromSlash(relative))
	if mkdirErr := os.MkdirAll(filepath.Dir(destination), 0o700); mkdirErr != nil {
		return fmt.Errorf("prepare R2 object path %s: %w", relative, mkdirErr)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create R2 object snapshot %s: %w", relative, err)
	}
	_, copyErr := io.Copy(file, output.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("snapshot R2 object %s: %w", relative, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close R2 object snapshot %s: %w", relative, closeErr)
	}

	return nil
}

func applyPlan(ctx context.Context, client s3Client, request Request, plan localrepo.SyncPlan) error {
	for _, action := range plan.Actions {
		key := request.Prefix + action.Path
		if action.Kind == "delete" {
			if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(request.Bucket),
				Key:    aws.String(key),
			}); err != nil {
				return fmt.Errorf("delete R2 object %s: %w", action.Path, err)
			}
			continue
		}
		if err := putObject(ctx, client, request, key, action.Path); err != nil {
			return err
		}
	}

	return nil
}

func putObject(
	ctx context.Context,
	client s3Client,
	request Request,
	key string,
	relative string,
) (returnErr error) {
	filePath := filepath.Join(request.Root, filepath.FromSlash(relative))
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open candidate object %s: %w", relative, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			returnErr = errors.Join(
				returnErr,
				fmt.Errorf("close candidate object %s: %w", relative, closeErr),
			)
		}
	}()
	digest, err := fileDigest(file)
	if err != nil {
		return fmt.Errorf("digest candidate object %s: %w", relative, err)
	}
	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
		return fmt.Errorf("rewind candidate object %s: %w", relative, seekErr)
	}
	if _, putErr := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(request.Bucket),
		Key:          aws.String(key),
		Body:         file,
		CacheControl: aws.String(request.cacheControl(relative)),
		Metadata:     map[string]string{"sha256": digest},
	}); putErr != nil {
		return fmt.Errorf("upload R2 object %s: %w", relative, putErr)
	}

	return nil
}

func (request Request) cacheControl(relative string) string {
	if !request.ProductionRoot {
		return "no-store"
	}
	if strings.HasSuffix(relative, ".deb") || strings.HasSuffix(relative, ".rpm") ||
		strings.Contains(relative, "/by-hash/") || isImmutableRPMMetadata(relative) {
		return "public, max-age=31536000, immutable"
	}

	return "no-store"
}

func isImmutableRPMMetadata(relative string) bool {
	if !strings.Contains(relative, "/repodata/") {
		return false
	}
	name := path.Base(relative)

	return name != "repomd.xml" && name != "repomd.xml.asc"
}

func fileDigest(file *os.File) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
