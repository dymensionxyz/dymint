package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SubmitBlobResponse represents the response from the Walrus publisher when submitting a blob
type SubmitBlobResponse struct {
	NewlyCreated struct {
		BlobObject struct {
			BlobID string `json:"blobId"`
		} `json:"blob_object"`
	} `json:"newlyCreated"`
	AlreadyCertified struct {
		BlobID string `json:"blob_id"`
	} `json:"alreadyCertified"`
}

// Client handles HTTP interactions with the Walrus API
type Client struct {
	publisherURL  string
	aggregatorURL string
	httpClient    *http.Client
}

// NewClient creates a new Walrus client instance
func NewClient(publisherUrl, aggregatorUrl string) *Client {
	return &Client{
		publisherURL:  publisherUrl,
		aggregatorURL: aggregatorUrl,
		httpClient:    &http.Client{},
	}
}

// SubmitBlob submits a blob to the Walrus publisher
func (c *Client) SubmitBlob(ctx context.Context, data []byte, storeDurationEpochs int, blobOwnerAddr string) (string, error) {
	url := fmt.Sprintf("%s/v1/blobs?epochs=%d&send_object_to=%s", c.publisherURL, storeDurationEpochs, blobOwnerAddr)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result SubmitBlobResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	blobID := result.NewlyCreated.BlobObject.BlobID
	if blobID == "" {
		blobID = result.AlreadyCertified.BlobID
	}
	if blobID == "" {
		return "", fmt.Errorf("no blob ID in response")
	}

	return blobID, nil
}

// GetBlob retrieves a blob from the Walrus aggregator
func (c *Client) GetBlob(ctx context.Context, blobID string) ([]byte, error) {
	url := fmt.Sprintf("%s/v1/blobs/%s", c.aggregatorURL, blobID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// CheckBlobAvailability checks if a blob is available on the Walrus aggregator
func (c *Client) CheckBlobAvailability(ctx context.Context, blobID string) error {
	url := fmt.Sprintf("%s/v1/blobs/%s", c.aggregatorURL, blobID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
