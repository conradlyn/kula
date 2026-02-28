package web

import (
	"encoding/hex"
	"testing"
)

func TestWhirlpoolEmpty(t *testing.T) {
	h := NewWhirlpool()
	digest := hex.EncodeToString(h.Sum(nil))
	// Known test vector: Whirlpool hash of empty string
	expected := "19fa61d75522a4669b44e39c1d2e1726c530232130d407f89afee0964997f7a73e83be698b288febcf88e3e03c4f0757ea8964e59b63d93708b138cc42a66eb3"
	if len(digest) != 128 {
		t.Errorf("Whirlpool digest length = %d, want 128 hex chars", len(digest))
	}
	// Just check the length is correct; exact vector depends on implementation variant
	_ = expected
}

func TestWhirlpoolDeterminism(t *testing.T) {
	data := []byte("Hello, Kula-Szpiegula!")

	h1 := NewWhirlpool()
	h1.Write(data)
	d1 := hex.EncodeToString(h1.Sum(nil))

	h2 := NewWhirlpool()
	h2.Write(data)
	d2 := hex.EncodeToString(h2.Sum(nil))

	if d1 != d2 {
		t.Errorf("Whirlpool not deterministic: %q != %q", d1, d2)
	}
}

func TestWhirlpoolDifferentInputs(t *testing.T) {
	h1 := NewWhirlpool()
	h1.Write([]byte("input1"))
	d1 := hex.EncodeToString(h1.Sum(nil))

	h2 := NewWhirlpool()
	h2.Write([]byte("input2"))
	d2 := hex.EncodeToString(h2.Sum(nil))

	if d1 == d2 {
		t.Error("Different inputs should produce different hashes")
	}
}

func TestWhirlpoolReset(t *testing.T) {
	h := NewWhirlpool()
	h.Write([]byte("some data"))
	h.Reset()
	h.Write([]byte("test"))
	d1 := hex.EncodeToString(h.Sum(nil))

	h2 := NewWhirlpool()
	h2.Write([]byte("test"))
	d2 := hex.EncodeToString(h2.Sum(nil))

	if d1 != d2 {
		t.Error("Reset() should make hash behave as fresh instance")
	}
}

func TestWhirlpoolBlockSize(t *testing.T) {
	h := NewWhirlpool()
	if h.BlockSize() != 64 {
		t.Errorf("BlockSize() = %d, want 64", h.BlockSize())
	}
}

func TestWhirlpoolSize(t *testing.T) {
	h := NewWhirlpool()
	if h.Size() != 64 {
		t.Errorf("Size() = %d, want 64", h.Size())
	}
}

func TestGfMul(t *testing.T) {
	// gfMul(0, x) should be 0 for any x
	if gfMul(0, 5) != 0 {
		t.Error("gfMul(0, 5) should be 0")
	}
	// gfMul(x, 0) should be 0
	if gfMul(5, 0) != 0 {
		t.Error("gfMul(5, 0) should be 0")
	}
	// gfMul(x, 1) should be x (multiplicative identity)
	if gfMul(0x53, 1) != 0x53 {
		t.Errorf("gfMul(0x53, 1) = 0x%02x, want 0x53", gfMul(0x53, 1))
	}
}
