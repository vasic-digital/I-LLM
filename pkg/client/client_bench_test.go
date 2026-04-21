package client

import (
	"context"
	"testing"
)

func BenchmarkRenderPattern(b *testing.B) {
	c, err := New()
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()
	p, err := c.GetPattern(context.Background(), "few-shot")
	if err != nil {
		b.Fatal(err)
	}
	vars := map[string]string{"examples": "E", "task": "T"}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.RenderPattern(ctx, *p, vars); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChainOfThought(b *testing.B) {
	c, err := New()
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		if _, err := c.ChainOfThought(ctx, "problem", "model"); err != nil {
			b.Fatal(err)
		}
	}
}
