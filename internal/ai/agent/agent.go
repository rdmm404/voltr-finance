package agent

import "context"

type Agent[I any, O any] interface {
	Run(ctx context.Context, input *I, mode StreamingMode) (<-chan *O, error)
}
