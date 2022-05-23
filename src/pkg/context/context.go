package context

import (
	"fmt"
	"net/http"
	"os"
)

// Context represents the infromation provided by the runtime, such as FC, docker container and etc.
type Context struct {
	AccountId       string
	AccessKeyId     string
	AccessKeySecret string
	SecurityToken   string
	Region          string
}

// Init the context from FC context.
func New(request *http.Request) (*Context, error) {
	ctx := &Context{
		AccessKeyId:     request.Header.Get("x-fc-access-key-id"),
		AccessKeySecret: request.Header.Get("x-fc-access-key-secret"),
		SecurityToken:   request.Header.Get("x-fc-security-token"),
		Region:          request.Header.Get("x-fc-region"),
	}

	if ctx.AccessKeyId == "" {
		return nil, fmt.Errorf("can not get access key id from fc runtime. Please make sure you already granted OSS permission to your FC function")
	}
	if ctx.AccessKeySecret == "" {
		return nil, fmt.Errorf("can not get access key secret from fc runtime. Please make sure you already granted OSS permission to your FC function")
	}
	if ctx.Region == "" {
		return nil, fmt.Errorf("can not get region from fc runtime. Please make sure you already granted OSS permission to your FC function")
	}

	return ctx, nil
}

// For test only.
func NewFromEnvVars() (*Context, error) {
	ctx := &Context{
		AccessKeyId:     os.Getenv("ALI_KEY_ID"),
		AccessKeySecret: os.Getenv("ALI_KEY_SECRET"),
		SecurityToken:   os.Getenv("ALI_SECURITY_TOKEN"),
		Region:          os.Getenv("ALI_REGION"),
	}

	if ctx.AccessKeyId == "" {
		return nil, fmt.Errorf("can not get access key id from environment variable")
	}
	if ctx.AccessKeySecret == "" {
		return nil, fmt.Errorf("can not get access key secret from environment variable")
	}
	if ctx.Region == "" {
		return nil, fmt.Errorf("can not get region from environment variable")
	}

	return ctx, nil
}
