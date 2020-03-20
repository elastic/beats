package reviewdog

import "context"

var _ BulkCommentService = &multiCommentService{}

type multiCommentService struct {
	services []CommentService
}

func (m *multiCommentService) Post(ctx context.Context, c *Comment) error {
	for _, cs := range m.services {
		if err := cs.Post(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

func (m *multiCommentService) Flush(ctx context.Context) error {
	for _, cs := range m.services {
		if bulk, ok := cs.(BulkCommentService); ok {
			if err := bulk.Flush(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// MultiCommentService creates a comment service that duplicates its post to
// all the provided comment services.
func MultiCommentService(services ...CommentService) CommentService {
	s := make([]CommentService, len(services))
	copy(s, services)
	return &multiCommentService{services: s}
}
