package httpapi

import "testing"

func TestOrderEmbeddingData(t *testing.T) {
	first, second := 0, 1
	data := []embeddingData{
		{Index: &second, Embedding: []float32{2}},
		{Index: &first, Embedding: []float32{1}},
	}
	if err := orderEmbeddingData(data); err != nil {
		t.Fatal(err)
	}
	if data[0].Embedding[0] != 1 || data[1].Embedding[0] != 2 {
		t.Fatalf("embeddings not restored to input order: %+v", data)
	}
}

func TestOrderEmbeddingDataRejectsIncompleteIndexes(t *testing.T) {
	index := 0
	data := []embeddingData{{Index: &index}, {}}
	if err := orderEmbeddingData(data); err == nil {
		t.Fatal("incomplete response indexes accepted")
	}
}
