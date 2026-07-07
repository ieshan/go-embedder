package embedder

import "context"

// FakeEmbedder is a test embedder that returns deterministic embeddings
// based on the input text hash. It does not make any external calls.
type FakeEmbedder struct {
	Dimensions int
}

// NewFakeEmbedder creates a FakeEmbedder with the given dimensions.
func NewFakeEmbedder(dimensions int) *FakeEmbedder {
	return &FakeEmbedder{Dimensions: dimensions}
}

func (f *FakeEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	return f.embed(text), nil
}

func (f *FakeEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		result[i] = f.embed(text)
	}
	return result, nil
}

func (f *FakeEmbedder) EmbedBatchPartial(_ context.Context, texts []string) ([][]float32, []error) {
	result := make([][]float32, len(texts))
	errs := make([]error, len(texts))
	for i, text := range texts {
		result[i] = f.embed(text)
	}
	return result, errs
}

func (f *FakeEmbedder) Dim() int { return f.Dimensions }

func (f *FakeEmbedder) embed(text string) []float32 {
	vec := make([]float32, f.Dimensions)
	var hash uint32 = 2166136261
	for _, b := range []byte(text) {
		hash ^= uint32(b)
		hash *= 16777619
	}
	for i := range f.Dimensions {
		hash ^= uint32(i)
		hash *= 16777619
		vec[i] = float32(hash%1000) / 1000.0
	}
	return vec
}
