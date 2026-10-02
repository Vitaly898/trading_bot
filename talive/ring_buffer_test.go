package talive

import (
	"testing"
)

func TestRingBufferCapacity(t *testing.T) {
	capacity := 2
	rb := newRingBuffer(capacity)
	for i := 0; i < 20; i++ {
		rb.Push(float64(i))
		if cap(rb.buffer) > capacity {
			t.Errorf("buffer capacity %d reached limit %d", cap(rb.buffer), capacity)
			break
		}
	}
}

func TestRingBuffer_Sum(t *testing.T) {
	rb := newRingBuffer(5)
	for i := 1; i <= 6; i++ {
		rb.Push(float64(i))
	}
	if rb.Sum != 20 {
		t.Errorf("wrong Sum value %f, expected 20.0", rb.Sum)
	}
}

func TestRingBuffer_SumExceptLast(t *testing.T) {
	rb := newRingBuffer(5)
	for i := 1; i <= 6; i++ {
		rb.Push(float64(i))
	}
	if rb.SumExceptLast() != 18 {
		t.Errorf("wrong SumExceptLast value %f, expected 18.0", rb.SumExceptLast())
	}
}

func TestRingBuffer_Min(t *testing.T) {
	rb := newRingBuffer(2)

	rb.Push(float64(3)) // removed
	rb.Push(float64(2)) // removed
	rb.Push(float64(1))
	rb.Push(float64(2))

	if rb.Min() != 1 {
		t.Errorf("wrong Min value %f, expected 1.0", rb.Min())
	}
}

func TestRingBuffer_MinExceptLast(t *testing.T) {
	rb := newRingBuffer(2)

	rb.Push(float64(3)) // removed
	rb.Push(float64(2)) // removed
	rb.Push(float64(1)) // next to rewrite - "last"
	rb.Push(float64(2))

	if rb.MinExceptLast() != 2 {
		t.Errorf("wrong Min value %f, expected 2.0", rb.MinExceptLast())
	}
}

func TestRingBuffer_Max(t *testing.T) {
	rb := newRingBuffer(2)
	rb.Push(float64(1)) // removed
	rb.Push(float64(2)) // removed
	rb.Push(float64(3))
	rb.Push(float64(2))

	if rb.Max() != 3 {
		t.Errorf("wrong Max value %f, expected 3.0", rb.Max())
	}
}

func TestRingBuffer_MaxExceptLast(t *testing.T) {
	rb := newRingBuffer(2)
	rb.Push(float64(1)) // removed
	rb.Push(float64(2)) // removed
	rb.Push(float64(3)) // next to rewrite - "last"
	rb.Push(float64(2))

	if rb.MaxExceptLast() != 2 {
		t.Errorf("wrong MaxExceptLast value %f, expected 2.0", rb.MaxExceptLast())
	}
}

func TestRingBuffer_MinMax(t *testing.T) {
	rb := newRingBuffer(3)
	rb.Push(float64(3)) // removed
	rb.Push(float64(2)) // removed
	rb.Push(float64(1))
	rb.Push(float64(2))
	rb.Push(float64(3))

	minV, maxV := rb.MinMax()
	if minV != 1 {
		t.Errorf("wrong MinMax min value %f, expected 1.0", minV)
	}
	if maxV != 3 {
		t.Errorf("wrong MinMax max value %f, expected 3.0", maxV)
	}
}

func TestRingBuffer_MinMaxExceptLast(t *testing.T) {
	rb := newRingBuffer(3)
	rb.Push(float64(3)) // removed
	rb.Push(float64(2)) // removed
	rb.Push(float64(1)) // next to rewrite - "last"
	rb.Push(float64(2))
	rb.Push(float64(3))

	minV, maxV := rb.MinMaxExceptLast()
	if minV != 2 {
		t.Errorf("wrong MinMaxExceptLast min value %f, expected 3.0", minV)
	}
	if maxV != 3 {
		t.Errorf("wrong MinMaxExceptLast max value %f, expected 3.0", maxV)
	}
}

func TestRingBuffer_Len(t *testing.T) {
	rb := newRingBuffer(2)
	expected := []int{0, 1, 2, 2}
	for i := 0; i < 4; i++ {
		if rb.Len() != expected[i] {
			t.Errorf("wrong Len %d at step %d, expected %d", rb.Len(), i, expected[i])
		}
		rb.Push(float64(i + 1))
	}
}

func TestRingBuffer_ReadersNoPanicOnPartialFill(_ *testing.T) {
	capacity := 5
	rb := newRingBuffer(capacity)
	for i := 0; i < capacity; i++ {
		_ = rb.Sum
		rb.SumExceptLast()
		rb.Last()
		rb.Min()
		rb.MinExceptLast()
		rb.Max()
		rb.MaxExceptLast()
		rb.MinMax()
		rb.MinMaxExceptLast()
		rb.Push(float64(i + 1))
	}
}
