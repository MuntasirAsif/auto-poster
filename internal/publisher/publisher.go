package publisher

import "context"

type Platform string

const (
	PlatformTwitter  Platform = "twitter"
	PlatformLinkedIn Platform = "linkedin"
	PlatformBlog     Platform = "blog"
)

type Publisher interface {
	Platform() Platform
	Publish(ctx context.Context, content string) error
}