package embedder

import "testing"

func TestFakeEmbedder(t *testing.T) {
	e := NewFakeEmbedder(128)
	if got := e.Dim(); got != 128 {
		t.Fatalf("Dim() = %d, want 128", got)
	}

	ctx := t.Context()

	t.Run("Embed", func(t *testing.T) {
		vec, err := e.Embed(ctx, "hello world")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vec) != 128 {
			t.Fatalf("vector length: want 128, got %d", len(vec))
		}
	})

	t.Run("EmbedBatch", func(t *testing.T) {
		texts := []string{"hello", "world", "foo"}
		vecs, err := e.EmbedBatch(ctx, texts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vecs) != 3 {
			t.Fatalf("vectors length: want 3, got %d", len(vecs))
		}
		for i, v := range vecs {
			if len(v) != 128 {
				t.Errorf("vectors[%d] length: want 128, got %d", i, len(v))
			}
		}
	})

	t.Run("EmbedBatchPartial", func(t *testing.T) {
		texts := []string{"a", "b"}
		vecs, errs := e.EmbedBatchPartial(ctx, texts)
		if len(vecs) != 2 {
			t.Fatalf("vectors length: want 2, got %d", len(vecs))
		}
		if len(errs) != 2 {
			t.Fatalf("errors length: want 2, got %d", len(errs))
		}
		for i, err := range errs {
			if err != nil {
				t.Errorf("errs[%d] = %v, want nil", i, err)
			}
		}
	})

	t.Run("Deterministic", func(t *testing.T) {
		v1, _ := e.Embed(ctx, "test")
		v2, _ := e.Embed(ctx, "test")
		for i := range v1 {
			if v1[i] != v2[i] {
				t.Fatalf("embeddings not deterministic: v1[%d]=%v, v2[%d]=%v", i, v1[i], i, v2[i])
			}
		}
	})
}
