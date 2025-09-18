package tool

import (
	"fmt"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mitchellh/mapstructure"
)

func createToolDecoder(output interface{}, hooks []mapstructure.DecodeHookFuncType) (*mapstructure.Decoder, error) {
	config := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           output,
	}

	if len(hooks) == 1 {
		config.DecodeHook = hooks[0]
	}

	if len(hooks) > 1 {
		config.DecodeHook = mapstructure.ComposeDecodeHookFunc(hooks)
	}

	decoder, err := mapstructure.NewDecoder(config)

	if err != nil {
		return nil, fmt.Errorf("error while creating decoder for tool - %w", err)
	}

	return decoder, nil
}

func dateToPgTimestampHook(from reflect.Type, to reflect.Type, data any) (any, error) {
	if to != reflect.TypeOf(pgtype.Timestamptz{}) {
		return data, nil
	}

	if data == nil {
		return pgtype.Timestamptz{Valid: false}, nil
	}

	switch v := data.(type) {
	case pgtype.Timestamp:
		return v, nil
	case time.Time:
		return pgtype.Timestamptz{Time: v, Valid: true}, nil
	case string:
		parsedTime, err := time.Parse("2006-01-02T15:04:05", v)
		fmt.Printf("hook: string: parsed time %v\n", parsedTime)
		if err != nil {
			return nil, fmt.Errorf("error while parsing string timestamp - %w", err)
		}

		return pgtype.Timestamptz{Time: parsedTime, Valid: true}, nil
	default:
		return nil, fmt.Errorf("unsupported data type")
	}
}
