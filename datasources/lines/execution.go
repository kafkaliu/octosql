package lines

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"time"

	. "github.com/cube2222/octosql/execution"
	"github.com/cube2222/octosql/octosql"
	"github.com/cube2222/octosql/physical"
)

type DatasourceExecuting struct {
	path, separator string
	fields          []physical.SchemaField
}

func (d *DatasourceExecuting) Run(ctx ExecutionContext, produce ProduceFn, metaSend MetaSendFn) error {
	f, err := os.Open(d.path)
	if err != nil {
		return fmt.Errorf("couldn't open file: %w", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if d.separator != "\n" {
		// Mostly copied from bufio.ScanLines.
		sc.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}
			if i := bytes.Index(data, []byte(d.separator)); i >= 0 {
				// We have a full separator-terminated line.
				return i + 1, data[0:i], nil
			}
			// If we're at EOF, we have a final, non-terminated line. Return it.
			if atEOF {
				return len(data), data, nil
			}
			// Request more data.
			return 0, nil, nil
		})
	}

	line := 0
	for sc.Scan() {
		values := make([]octosql.Value, len(d.fields))
		for i := range d.fields {
			switch d.fields[i].Name {
			case "number":
				values[i] = octosql.NewInt(line)
			case "text":
				values[i] = octosql.NewString(sc.Text())
			}
		}

		if err := produce(ProduceFromExecutionContext(ctx), NewRecord(values, false, time.Time{})); err != nil {
			return fmt.Errorf("couldn't produce record: %w", err)
		}
		line++
	}
	if sc.Err() != nil {
		return err
	}
	return nil
}
