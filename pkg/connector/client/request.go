package enterprise

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

func toValues(queryParameters map[string]interface{}) string {
	params := url.Values{}
	for key, valueAny := range queryParameters {
		switch value := valueAny.(type) {
		case string:
			params.Add(key, value)
		case int:
			params.Add(key, strconv.Itoa(value))
		case bool:
			params.Add(key, strconv.FormatBool(value))
		case []string:
			// Handle string arrays for Slack API
			for _, str := range value {
				params.Add(key, str)
			}
		default:
			continue
		}
	}
	return params.Encode()
}

func (c *Client) getUrl(
	path string,
	queryParameters map[string]interface{},
	useScim bool,
) *url.URL {
	var output *url.URL
	if useScim {
		output = c.baseScimUrl.JoinPath(path)
	} else {
		output = c.baseUrl.JoinPath(path)
	}

	output.RawQuery = toValues(queryParameters)
	return output
}

// WithBearerToken - TODO(marcos): move this function to `baton-sdk`.
func WithBearerToken(token string) uhttp.RequestOption {
	return uhttp.WithHeader("Authorization", fmt.Sprintf("Bearer %s", token))
}

func (c *Client) post(
	ctx context.Context,
	path string,
	target interface{},
	payload map[string]interface{},
	useBotToken bool,
) (
	*v2.RateLimitDescription,
	error,
) {
	token := c.token
	if useBotToken {
		token = c.botToken
	}

	return c.doRequest(
		ctx,
		http.MethodPost,
		c.getUrl(path, nil, false),
		target,
		WithBearerToken(token),
		uhttp.WithFormBody(toValues(payload)),
	)
}

func (c *Client) postJSON(
	ctx context.Context,
	path string,
	target interface{},
	payload interface{},
	useBotToken bool,
) (
	*v2.RateLimitDescription,
	error,
) {
	token := c.token
	if useBotToken {
		token = c.botToken
	}

	return c.doRequest(
		ctx,
		http.MethodPost,
		c.getUrl(path, nil, false),
		target,
		WithBearerToken(token),
		uhttp.WithJSONBody(payload),
	)
}

func (c *Client) doRequest(
	ctx context.Context,
	method string,
	url *url.URL,
	target interface{},
	options ...uhttp.RequestOption,
) (
	*v2.RateLimitDescription,
	error,
) {
	logger := ctxzap.Extract(ctx)
	logger.Debug(
		"making request",
		zap.String("method", method),
		zap.String("url", url.String()),
	)

	options = append(
		options,
		uhttp.WithAcceptJSONHeader(),
	)

	request, err := c.wrapper.NewRequest(
		ctx,
		method,
		url,
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	var ratelimitData v2.RateLimitDescription
	response, err := c.wrapper.Do(
		request,
		uhttp.WithRatelimitData(&ratelimitData),
	)
	if err != nil {
		if response != nil && response.Body != nil {
			bodyBytes, _ := io.ReadAll(response.Body)
			logger.Error("request failed", zap.String("body", logBody(bodyBytes, 2048)))
		}
		return &ratelimitData, err
	}
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Error("failed to read response body", zap.String("body", logBody(bodyBytes, 2048)))
		return &ratelimitData, fmt.Errorf("failed to read response body: %w", err)
	}

	if response.StatusCode == http.StatusNoContent || len(bodyBytes) == 0 {
		return &ratelimitData, nil
	}

	if err := json.Unmarshal(bodyBytes, &target); err != nil {
		logger.Error("failed to unmarshal response", zap.String("body", logBody(bodyBytes, 2048)))
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.StatusCode == http.StatusOK {
		if err := ErrorWithGrpcCodeFromBytes(bodyBytes); err != nil {
			return &ratelimitData, err
		}
	}

	return &ratelimitData, nil
}
