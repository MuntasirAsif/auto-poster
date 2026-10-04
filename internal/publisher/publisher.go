package publisher

import "context"

type Platform string

const (
	PlatformTwitter  Platform = "twitter"
	PlatformLinkedIn Platform = "linkedin"
	PlatformBlog     Platform = "blog"
)

type PublishRequest struct {
	Title            string
	Content          string
	Category         string
	ShortDescription string
	Image            []byte
}

type Publisher interface {
	Platform() Platform
	Publish(ctx context.Context, req PublishRequest) (string, error)
}