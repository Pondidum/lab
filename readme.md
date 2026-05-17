# Lab

> Run experiments in your codebase.

## Example

```go
func Render() string {

	list, err := RunExperiment(
		ctx,
		WithBaseline(func(ctx context.Context) (string, error) {
			return renderList(things) // the old way
		}),
		WithTrial(func(ctx context.Context) (string, error) {
			return renderListTpl(things) // the new way
		}),
		WithComparer(func(ctx context.Context, baseline, trial *Observation[thingy]) {
			// trace what has happened.

			attrs := []attribute.Attribute{
				attribute.String("experiment.name", "replace_html_list"),
				attribute.Boolean("experiment.passed", baseline.Result == experiment.Result),
			}

			if baseline.Err != nil {
				attrs := append(attrs, attribute.String("experiment.baseline.err", baseline.Err))
			}
			if experiment.Err != nil {
				attrs := append(attrs, attribute.String("experiment.trial.err", experiment.Err))
			}

			_, span := tr.StartSpan(ctx, "experiment.comparison")
			span.SetAttributes(attrs)
			span.End()
		}),
	)
	if err != nil {
		return err.String()
	}

	html := header()
	html += body(list)
	html += footer()

	return string
}
```

- Runs both the `Baseline` and the `Trial`
- Returns results from `Baseline` ONLY
- Catches errors and panics from the `Trial`s
- Exposes this to the `Comparer` so you can record what has happened
