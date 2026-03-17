package cost

// ModelPricing holds per-million-token costs for a model.
type ModelPricing struct {
	InputPerMTok  float64
	OutputPerMTok float64
}

// defaultPricing is the embedded pricing table. Update values as provider pricing changes.
var defaultPricing = map[string]ModelPricing{
	"claude-sonnet-4-20250514":  {InputPerMTok: 3.0, OutputPerMTok: 15.0},
	"claude-haiku-4-5-20251001": {InputPerMTok: 0.80, OutputPerMTok: 4.0},
	"claude-opus-4-20250514":    {InputPerMTok: 15.0, OutputPerMTok: 75.0},
	"gpt-4o":                    {InputPerMTok: 2.50, OutputPerMTok: 10.0},
	"gpt-4o-mini":               {InputPerMTok: 0.15, OutputPerMTok: 0.60},
	"gpt-4.1":                   {InputPerMTok: 2.0, OutputPerMTok: 8.0},
	"gpt-4.1-mini":              {InputPerMTok: 0.40, OutputPerMTok: 1.60},
	"gpt-4.1-nano":              {InputPerMTok: 0.10, OutputPerMTok: 0.40},
}

// LookupPrice returns the per-million-token pricing for the given model.
// Returns zero values if the model is not found.
func LookupPrice(model string) (inputPerMTok, outputPerMTok float64) {
	p, ok := defaultPricing[model]
	if !ok {
		return 0, 0
	}
	return p.InputPerMTok, p.OutputPerMTok
}
