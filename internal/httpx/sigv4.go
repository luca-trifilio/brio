package httpx

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// signAWSv4 signs req in-place using the provided creds.
// The body is consumed and replaced with a fresh ReadCloser so the request
// remains executable afterwards.
func signAWSv4(req *http.Request, creds *AWSCreds) error {
	if creds == nil {
		return fmt.Errorf("awsv4: missing credentials")
	}
	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return fmt.Errorf("awsv4: empty access key or secret")
	}
	service := creds.Service
	if service == "" {
		service = "execute-api"
	}
	region := creds.Region
	if region == "" {
		region = "us-east-1"
	}

	// Compute payload hash from body.
	var bodyBytes []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("awsv4: read body: %w", err)
		}
		_ = req.Body.Close()
		bodyBytes = b
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	}
	sum := sha256.Sum256(bodyBytes)
	payloadHash := hex.EncodeToString(sum[:])

	awsCreds := aws.Credentials{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
	}

	signer := v4.NewSigner()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := signer.SignHTTP(ctx, awsCreds, req, payloadHash, service, region, time.Now().UTC()); err != nil {
		return fmt.Errorf("awsv4: sign: %w", err)
	}
	return nil
}
