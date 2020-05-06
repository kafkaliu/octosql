package streaming

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/cube2222/octosql"
	"github.com/cube2222/octosql/execution"
	"github.com/cube2222/octosql/storage"
)

type InstantStreamOutput struct {
}

func (o *InstantStreamOutput) ReadyForMore(ctx context.Context, tx storage.StateTransaction) error {
	return nil
}

func (o *InstantStreamOutput) AddRecord(ctx context.Context, tx storage.StateTransaction, inputIndex int, record *execution.Record) error {
	if inputIndex != 0 {
		return errors.Errorf("only one input stream allowed for output, got input index %d", inputIndex)
	}

	outputRecords := execution.NewOutputQueue(tx.WithPrefix(outputRecordsPrefix))
	if err := outputRecords.Push(ctx, record); err != nil {
		return errors.Wrap(err, "couldn't push current record to output queue")
	}

	return nil
}

func (o *InstantStreamOutput) Next(ctx context.Context, tx storage.StateTransaction) (*execution.Record, error) {
	// If no element to get and end of stream then end of stream else wait. Also check error.
	records := execution.NewOutputQueue(tx.WithPrefix(outputRecordsPrefix))
	var record execution.Record
	err := records.Pop(ctx, &record)
	if execution.GetErrWaitForChanges(err) != nil {
		errWaitForChanges := err

		var value octosql.Value

		endOfStreamState := storage.NewValueState(tx.WithPrefix(endOfStreamPrefix))
		err := endOfStreamState.Get(&value)
		if err == nil {
			return nil, execution.ErrEndOfStream
		} else if err == storage.ErrNotFound {
		} else if err != nil {
			return nil, errors.Wrap(err, "couldn't check end of stream state")
		}

		errorState := storage.NewValueState(tx.WithPrefix(errorPrefix))
		err = errorState.Get(&value)
		if err == nil {
			return nil, errors.New(value.AsString())
		} else if err == storage.ErrNotFound {
		} else if err != nil {
			return nil, errors.Wrap(err, "couldn't check end of stream state")
		}

		return nil, errWaitForChanges
	} else if err != nil {
		return nil, errors.Wrap(err, "couldn't get record from records queue")
	}
	return &record, nil
}

func (o *InstantStreamOutput) UpdateWatermark(ctx context.Context, tx storage.StateTransaction, watermark time.Time) error {
	return nil
}

func (o *InstantStreamOutput) GetWatermark(ctx context.Context, tx storage.StateTransaction) (time.Time, error) {
	panic("not implemented")
}

func (o *InstantStreamOutput) MarkEndOfStream(ctx context.Context, tx storage.StateTransaction) error {
	endOfStreamState := storage.NewValueState(tx.WithPrefix(endOfStreamPrefix))

	phantom := octosql.MakePhantom()
	if err := endOfStreamState.Set(&phantom); err != nil {
		return errors.Wrap(err, "couldn't mark end of stream")
	}

	return nil
}

func (o *InstantStreamOutput) GetEndOfStream(ctx context.Context, tx storage.StateTransaction) (bool, error) {
	endOfStreamState := storage.NewValueState(tx.WithPrefix(endOfStreamPrefix))

	var octoEndOfStream octosql.Value
	err := endOfStreamState.Get(&octoEndOfStream)
	if err == storage.ErrNotFound {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "couldn't get end of stream value")
	}

	return true, nil
}

func (o *InstantStreamOutput) MarkError(ctx context.Context, tx storage.StateTransaction, err error) error {
	errorState := storage.NewValueState(tx.WithPrefix(errorPrefix))

	octoError := octosql.MakeString(err.Error())
	if err := errorState.Set(&octoError); err != nil {
		return errors.Wrap(err, "couldn't mark error")
	}

	return nil
}

func (o *InstantStreamOutput) GetErrorMessage(ctx context.Context, tx storage.StateTransaction) (string, error) {
	errorState := storage.NewValueState(tx.WithPrefix(errorPrefix))

	var octoError octosql.Value
	err := errorState.Get(&octoError)
	if err == storage.ErrNotFound {
		return "", nil
	} else if err != nil {
		return "", errors.Wrap(err, "couldn't get error message")
	}

	return octoError.AsString(), nil
}

func (o *InstantStreamOutput) Close(ctx context.Context, storage storage.Storage) error {
	return nil // TODO: Cleanup?
}
